package main

import (
	"flag"
	"net"

	"github.com/epiphyte/goutils"
)

func serve(bind string) {
	l, err := net.Listen("tcp", bind)
	if err != nil {
		goutils.WriteError("unable to listen", err)
		panic("unable to start server")
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			goutils.WriteError("unable to accept client", err)
			continue
		}
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 65535)
	n, err := conn.Read(buf)
	if err != nil {
		goutils.WriteError("unable to read", err)
		return
	}
	goutils.WriteDebug("content", string(buf[0:n]))
}

func main() {
	bind := flag.String("bind", "127.0.0.1:5000", "binding address")
	flag.Parse()
	serve(*bind)
}
