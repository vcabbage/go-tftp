// Copyright (C) 2016 Kale Blankenship. All rights reserved.
// This software may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/cheggaaa/pb.v1"

	"github.com/codegangsta/cli"
	"github.com/vcabbage/trivialt"
)

var (
	// VERSION is replaced by linker for release builds.
	VERSION = "dev"
)

func main() {
	log.SetFlags(0)

	clientOpts := []cli.Flag{
		cli.IntFlag{
			Name:  "blksize, b",
			Usage: "Number of data bytes to send per-packet.",
			Value: 512,
		},
		cli.IntFlag{
			Name:  "windowsize, w",
			Usage: "Number of packets to send before requiring an acknowledgement.",
			Value: 1,
		},
		cli.IntFlag{
			Name:  "timeout, t",
			Usage: "Number of seconds to wait before terminating a stalled connection.",
			Value: 10,
		},
		cli.BoolTFlag{
			Name:  "tsize",
			Usage: "Enable the transfer size option. (default)",
		},
		cli.IntFlag{
			Name:  "retransmit, r",
			Value: 10,
			Usage: "Maximum number of back-to-back lost packets before terminating the connection.",
		},
		cli.BoolFlag{
			Name:  "netascii",
			Usage: "Enable netascii transfer mode.",
		},
		cli.BoolTFlag{
			Name:  "binary, octet, i",
			Usage: "Enable binary transfer mode. (default)",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "Don't display progress.",
		},
	}

	app := cli.NewApp()
	app.Name = "trivialt"
	app.Usage = "tftp server/client"
	app.Version = VERSION
	app.Commands = []cli.Command{
		{
			Name:      "serve",
			Aliases:   []string{"s"},
			Usage:     "Serve files from the filesystem.",
			ArgsUsage: `[bind address] [root directory]`,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "writable, w",
					Usage: "Enable file upload.",
				},
				cli.BoolFlag{
					Name:  "single-port, sp",
					Usage: "Enable single port mode. [Experimental]",
				},
			},
			Description: `Serves files from the local file systemd.
   
   Bind address is in form "ip:port". Omitting the IP will listen on all interfaces.
   If not specified the server will listen on all interfaces, port 69.app
   
   Files will be served from root directory. If omitted files will be served from
   the current directory.`,
			Action: cmdServe,
		},
		{
			Name:      "get",
			Aliases:   []string{"g"},
			Usage:     "Download file from a server.",
			ArgsUsage: "[server:port] [file]",
			Flags: append(clientOpts, cli.StringFlag{
				Name: "output, o",
				Usage: `Sets the output location to write the file. If not specified the
                                file will be written in the current directory.
                                Specifying "-" will write the file to stdout. ("-" implies "--quiet")`,
			}),
			Action: cmdGet,
		},
		{
			Name:      "put",
			Aliases:   []string{"p"},
			Usage:     "Upload file to a server.",
			ArgsUsage: "[server:port] [file]",
			Flags:     clientOpts,
			Action:    cmdPut,
		},
	}
	app.Run(os.Args)
}

func cmdServe(c *cli.Context) {
	addr := c.Args().First()
	path := c.Args().Get(1)
	writable := c.Bool("writable")
	singlePort := c.Bool("single-port")

	if addr == "" {
		addr = ":69"
	}

	root, err := filepath.Abs(path)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Starting TFTP Server on %q, serving %q\n", addr, root)
	fs := &server{trivialt.FileServer(root)}

	server, err := trivialt.NewServer(addr, trivialt.ServerSinglePort(singlePort))
	if err != nil {
		log.Fatalln(err)
	}

	server.ReadHandler(fs)
	if writable {
		server.WriteHandler(fs)
	}

	log.Println(server.ListenAndServe())
}

type server struct {
	fs trivialt.ReadWriteHandler
}

func (s *server) ServeTFTP(r trivialt.ReadRequest) {
	log.Printf("Read Request from %s for %q", r.Addr(), r.Name())
	s.fs.ServeTFTP(r)
}

func (s *server) ReceiveTFTP(r trivialt.WriteRequest) {
	log.Printf("Write Request from %s for %q", r.Addr(), r.Name())
	s.fs.ReceiveTFTP(r)
}

func cmdGet(c *cli.Context) {
	server := c.Args().Get(0)
	filePath := c.Args().Get(1)
	quiet := c.Bool("quiet") || c.String("output") == "-"

	if quiet {
		log.SetOutput(ioutil.Discard)
	}

	client := newClient(c)
	resp, err := client.Get(server + "/" + filePath)
	if err != nil {
		log.Fatalln("\n", trivialt.ErrorCause(err))
	}

	var out io.Writer
	if output := c.String("output"); output == "-" {
		out = os.Stdout
	} else {
		if output == "" {
			output = filepath.Base(filePath)
		}
		file, err := os.Create(output)
		if err != nil {
			log.Fatalln(err)
		}
		defer file.Close()
		out = file
	}

	var in io.Reader = resp
	if !quiet {
		size, err := resp.Size()
		if err != nil {
			log.Println("No size received from server.")
		}
		log.Println(filePath + ":")
		progress, finish := newProgresReader(resp, size)
		defer finish()
		in = progress
	}

	if _, err = io.Copy(out, in); err != nil {
		log.Fatalln("\n", trivialt.ErrorCause(err))
	}
}

func cmdPut(c *cli.Context) {
	client := newClient(c)

	server := c.Args().Get(0)
	filePath := c.Args().Get(1)
	quiet := c.Bool("quiet")

	if quiet {
		log.SetOutput(ioutil.Discard)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	var filesize int64
	var tsize int64
	if finfo, err := file.Stat(); err == nil {
		filesize = finfo.Size() // Set filesize for progress
		if c.Bool("tsize") {
			tsize = filesize // Set tsize if option enabled
		}
	}

	filename := filepath.Base(filePath)

	var upload io.Reader = file
	if !quiet {
		fmt.Println(filename + ":")
		progress, finish := newProgresReader(upload, filesize)
		defer finish()
		upload = progress
	}

	url := server + "/" + filename
	if err = client.Put(url, upload, tsize); err != nil {
		log.Fatalln("\n", trivialt.ErrorCause(err))
	}
}

func newClient(c *cli.Context) *trivialt.Client {
	mode := trivialt.ModeOctet
	if c.Bool("netascii") {
		mode = trivialt.ModeNetASCII
	}

	opts := []trivialt.ClientOpt{
		trivialt.ClientBlocksize(c.Int("blksize")),
		trivialt.ClientTransferSize(c.Bool("tsize")),
		trivialt.ClientWindowsize(c.Int("windowsize")),
		trivialt.ClientTimeout(c.Int("timeout")),
		trivialt.ClientRetransmit(c.Int("retransmit")),
		trivialt.ClientMode(mode),
	}

	client, err := trivialt.NewClient(opts...)
	if err != nil {
		log.Println()
		log.Fatalln(err)
	}

	return client
}

func newProgresReader(r io.Reader, size int64) (io.Reader, func()) {
	progress := pb.New64(size).SetUnits(pb.U_BYTES)
	progress.SetMaxWidth(100)
	progress.ShowSpeed = true
	progress.ShowFinalTime = true
	progress.Output = os.Stderr
	progress.Start()
	progressRdr := progress.NewProxyReader(r)
	return progressRdr, progress.Finish
}
