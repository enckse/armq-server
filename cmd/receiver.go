package main

import (
	"flag"
	"net"

	"github.com/epiphyte/goutils"
)

func serve(bind string, debug bool) {
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
		go handleRequest(conn, debug)
	}
}

func payload(data string, debug bool) {
	if debug {
		goutils.WriteDebug("data payload", data)
	}
}

func handleRequest(conn net.Conn, debug bool) {
	defer conn.Close()
	buf := make([]byte, 65535)
	n, err := conn.Read(buf)
	if err != nil {
		goutils.WriteError("unable to read", err)
		return
	}
	go payload(string(buf[0:n]), debug)
}

func main() {
	bind := flag.String("bind", "127.0.0.1:5000", "binding address")
	debug := flag.Bool("debug", false, "enable debugging")
	flag.Parse()
	opts := goutils.NewLogOptions()
	opts.Debug = *debug
	goutils.ConfigureLogging(opts)
	serve(*bind, *debug)
}
