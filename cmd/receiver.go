package main

import (
	"bytes"
	"flag"
	"io"
	"net"
	"sync"

	"github.com/epiphyte/goutils"
)

var (
	indexing  bool = false
	indexLock      = &sync.Mutex{}
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
	go index()
}

func index() {
	returning := true
	indexLock.Lock()
	if !indexing {
		returning = false
		indexing = true
	}
	indexLock.Unlock()
	if returning {
		return
	}
	goutils.WriteDebug("indexing")
	indexLock.Lock()
	indexing = false
	indexLock.Unlock()
}

func handleRequest(conn net.Conn, debug bool) {
	defer conn.Close()
	buffer := bytes.Buffer{}
	datum := false
	for {
		buf := make([]byte, 65535)
		n, err := conn.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			goutils.WriteError("unable to read", err)
			break
		}
		datum = true
		buffer.Write(buf[0:n])
	}
	if datum {
		go payload(string(buffer.Bytes()), debug)
	}
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
