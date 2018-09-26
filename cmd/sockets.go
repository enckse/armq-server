package main

import (
	"github.com/epiphyte/goutils"
)

type socketReader struct {
	reader
}

func payload(data string, debug bool) {
	if debug {
		goutils.WriteDebug("data payload", data)
	}
}

type receiver struct {
	goutils.SocketReceive
	debug bool
}

func (r *receiver) Consume(d []byte) {
	payload(string(d), r.debug)
}

func socketReceiver(bind string, debug bool) {
	socket := goutils.SocketSettings()
	socket.Bind = bind
	onReceive := &receiver{}
	onReceive.debug = debug
	goutils.SocketReceiveOnly(socket, onReceive)
}
