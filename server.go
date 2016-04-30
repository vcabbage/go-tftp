// Copyright (C) 2016 Kale Blankenship. All rights reserved.
// This software may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details

package trivialt

import "net"

// Server contains the configuration to run a TFTP server.
//
// A ReadHandler, WriteHandler, or both can be registered to the server. If one
// of the handlers isn't registered, the server will return errors to clients
// attempting to use them.
type Server struct {
	log     *logger
	net     string
	addrStr string
	addr    *net.UDPAddr
	conn    *net.UDPConn
	close   bool

	retransmit int // Per-packet retransmission limit

	rh ReadHandler
	wh WriteHandler
}

// NewServer returns a configured Server.
//
// Addr is the network address to listen on and is in the form "host:port".
// If a no host is given the server will listen on all interfaces.
//
// Any number of ServerOpts can be provided to configure optional values.
func NewServer(addr string, opts ...ServerOpt) (*Server, error) {
	s := &Server{
		log:        newLogger("server"),
		net:        defaultUDPNet,
		addrStr:    addr,
		retransmit: defaultRetransmit,
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// Addr is the network address of the server. It is available
// after the server has been started.
func (s *Server) Addr() (*net.UDPAddr, error) {
	if s.conn == nil {
		return nil, ErrAddressNotAvailable
	}
	return s.conn.LocalAddr().(*net.UDPAddr), nil
}

// ReadHandler registers a ReadHandler for the server.
func (s *Server) ReadHandler(rh ReadHandler) {
	s.rh = rh
}

// WriteHandler registers a WriteHandler for the server.
func (s *Server) WriteHandler(wh WriteHandler) {
	s.wh = wh
}

// Serve starts the server using an existing UDPConn.
func (s *Server) Serve(conn *net.UDPConn) error {
	if s.rh == nil && s.wh == nil {
		return ErrNoRegisteredHandlers
	}

	s.conn = conn

	buf := make([]byte, 1024)
	for {
		numBytes, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if s.close {
				return nil
			}
			return wrapError(err, "reading from conn")
		}
		bufCopy := make([]byte, numBytes)
		copy(bufCopy, buf)
		go s.dispatchRequest(addr, bufCopy)
	}
}

// Close stops the server and closes the network connection.
func (s *Server) Close() error {
	s.close = true
	return s.conn.Close()
}

// dispatchRequest parses incoming requests and executes the corresponding
// handler, if one is registered. If a handler is not registered the server
// sends an error to the client.
func (s *Server) dispatchRequest(addr *net.UDPAddr, b []byte) {
	conn, err := newConn(s.net, "", addr) // Use empty mode until request has been parsed.
	if err != nil {
		s.log.err("Received error opening connection for new request: %v", err)
		return
	}
	defer errorDefer(conn.Close, s.log, "error closing network connection in dispath")

	// Set retransmit
	conn.retransmit = s.retransmit

	// Validate request datagram
	conn.rx.setBytes(b)
	if err := conn.rx.validate(); err != nil {
		s.log.debug("Error decoding new request: %v", err)
		return
	}
	s.log.debug("New request from %v: %s", addr, conn.rx)

	// Set mode from request
	conn.mode = conn.rx.mode()

	switch conn.rx.opcode() {
	case opCodeRRQ:
		// Check for handler
		if s.rh == nil {
			s.log.debug("No read handler registered.")
			conn.sendError(ErrCodeIllegalOperation, "Server does not support read requests.")
			return
		}

		// Create request
		w := &readRequest{conn: conn, name: conn.rx.filename()}

		// execute handler
		s.rh.ServeTFTP(w)
	case opCodeWRQ:
		// Check for handler
		if s.wh == nil {
			s.log.debug("No write handler registered.")
			conn.sendError(ErrCodeIllegalOperation, "Server does not support write requests.")
			return
		}

		// Create request
		w := &writeRequest{conn: conn, name: conn.rx.filename()}

		// parse options to get size
		conn.log.trace("performing write setup")
		if err := conn.readSetup(true); err != nil {
			conn.err = err
		}

		s.wh.ReceiveTFTP(w)
	default:
		s.log.debug("Unexpected Request")
	}
}

// ListenAndServe starts a configured server.
func (s *Server) ListenAndServe() error {
	addr, err := net.ResolveUDPAddr(s.net, s.addrStr)
	if err != nil {
		return wrapError(err, "resolving server address")
	}
	s.addr = addr

	conn, err := net.ListenUDP(s.net, s.addr)
	if err != nil {
		return wrapError(err, "opening network connection")
	}

	return wrapError(s.Serve(conn), "serving tftp")
}

// ListenAndServe creates and starts a Server with default options.
func ListenAndServe(addr string, rh ReadHandler, wh WriteHandler) error {
	s, err := NewServer(addr)
	if err != nil {
		return wrapError(err, "creaing new server")
	}

	s.rh = rh
	s.wh = wh

	return wrapError(s.ListenAndServe(), "starting server")
}

// ServerOpt is a function that configures a Server.
type ServerOpt func(*Server) error

// ServerNet configures the network a server listens on.
// Must be one of: udp, udp4, udp6.
//
// Default: udp.
func ServerNet(net string) ServerOpt {
	return func(s *Server) error {
		if net != "udp" && net != "udp4" && net != "udp6" {
			return ErrInvalidNetwork
		}
		s.net = net
		return nil
	}
}

// ServerRetransmit configures the per-packet retransmission limit for all requests.
//
// Default: 10.
func ServerRetransmit(i int) ServerOpt {
	return func(s *Server) error {
		if i < 0 {
			return ErrInvalidRetransmit
		}
		s.retransmit = i
		return nil
	}
}
