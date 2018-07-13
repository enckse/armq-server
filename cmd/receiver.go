package main

import (
	"flag"
	"sync"

	"github.com/epiphyte/goutils"
)

var (
	indexing  bool = false
	indexLock      = &sync.Mutex{}
)

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

type receiver struct {
	goutils.SocketReceive
	debug bool
}

func (r *receiver) Consume(d []byte) {
	payload(string(d), r.debug)
}

func main() {
	bind := flag.String("bind", "127.0.0.1:5000", "binding address")
	debug := flag.Bool("debug", false, "enable debugging")
	flag.Parse()
	opts := goutils.NewLogOptions()
	opts.Debug = *debug
	goutils.ConfigureLogging(opts)
	socket := goutils.SocketSettings()
	socket.Bind = *bind
	onReceive := &receiver{}
	onReceive.debug = *debug
	goutils.SocketReceiveOnly(socket, onReceive)
}
