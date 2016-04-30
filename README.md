# **trivialt**

trivialt is a cross-platform, concurrent TFTP server and client. It can be used as a standalone executable or included in a Go project as a library.

### Standards Implemented

- [X] Binary Transfer ([RFC 1350](https://tools.ietf.org/html/rfc1350))
- [X] Netascii Transfer ([RFC 1350](https://tools.ietf.org/html/rfc1350))
- [X] Option Extention ([RFC 2347](https://tools.ietf.org/html/rfc2347))
- [X] Blocksize Option ([RFC 2348](https://tools.ietf.org/html/rfc2348))
- [X] Timeout Interval Option ([RFC 2349](https://tools.ietf.org/html/rfc2349))
- [X] Transfer Size Option ([RFC 2349](https://tools.ietf.org/html/rfc2349))
- [X] Windowsize Options ([RFC 7440](https://tools.ietf.org/html/rfc7440))

## Installation

If you have the Go toolchain installed you can simply `go get` the packages. This will download the source into your `$GOPATH` and install the binary to `$GOPATH/bin/trivialt`.

``` bash
go get -u github.com/vcabbage/trivialt/...
```

Binary downloads coming soon.

## Command Usage

Running as a server:
```
# trivialt serve --help
NAME:
   trivialt serve - Serve files from the filesystem.

USAGE:
   trivialt serve [bind address] [root directory]

DESCRIPTION:
   Serves files from the local file systemd.

   Bind address is in form "ip:port". Omitting the IP will listen on all interfaces.
   If not specified the server will listen on all interfaces, port 69.app

   Files will be served from root directory. If omitted files will be served from
   the current directory.

OPTIONS:
   --writeable, -w	Enable file upload.
```

```
# trivialt serve :6900 /tftproot --writable
Starting TFTP Server on ":6900", serving "/tftproot"
Read Request from 127.0.0.1:61877 for "ubuntu-16.04-server-amd64.iso"
Write Request from 127.0.0.1:51205 for "ubuntu-16.04-server-amd64.iso"

```

Downloading a file:
```
# trivialt get --help
NAME:
   trivialt get - Download file from a server.

USAGE:
   trivialt get [command options] [server:port] [file]

OPTIONS:
   --blksize, -b "512"      Number of data bytes to send per-packet.
   --windowsize, -w "1"     Number of packets to send before requiring an acknowledgement.
   --timeout, -t "10"       Number of seconds to wait before terminating a stalled connection.
   --tsize                  Enable the transfer size option. (default)
   --retransmit, -r "10"    Maximum number of back-to-back lost packets before terminating the connection.
   --netascii               Enable netascii transfer mode.
   --binary, --octet, -i    Enable binary transfer mode. (default)
   --quiet, -q              Don't display progress.
   --output, -o             Sets the output location to write the file. If not specified the
                            file will be written in the current directory.
                            Specifying "-" will write the file to stdout. ("-" implies "--quiet")
```

```
# trivialt get localhost:6900 ubuntu-16.04-server-amd64.iso
ubuntu-16.04-server-amd64.iso:
 655.00 MB / 655.00 MB [=====================================================] 100.00% 16.76 MB/s39s
```

Uploading a file:
```
# trivialt get --help
NAME:
   trivialt get - Download file from a server.

USAGE:
   trivialt get [command options] [server:port] [file]

OPTIONS:
   --blksize, -b "512"      Number of data bytes to send per-packet.
   --windowsize, -w "1"     Number of packets to send before requiring an acknowledgement.
   --timeout, -t "10"       Number of seconds to wait before terminating a stalled connection.
   --tsize                  Enable the transfer size option. (default)
   --retransmit, -r "10"	Maximum number of back-to-back lost packets before terminating the connection.
   --netascii               Enable netascii transfer mode.
   --binary, --octet, -i	Enable binary transfer mode. (default)
   --quiet, -q              Don't display progress.
   --output, -o             Sets the output location to write the file. If not specified the
                            file will be written in the current directory.
                            Specifying "-" will write the file to stdout. ("-" implies "--quiet")
```

```
# trivialt put localhost:6900 ubuntu-16.04-server-amd64.iso --blksize 1468 --windowsize 16
ubuntu-16.04-server-amd64.iso:
 655.00 MB / 655.00 MB [=====================================================] 100.00% 178.41 MB/s3s
```