package main

import (
	"github.com/epiphyte/goutils"
)

type receiver struct {
	goutils.SocketReceive
}

func (r *receiver) Consume(d []byte) {
	goutils.WriteInfo(string(d))
	queue("", d, false)
}

func socketReceiver(ctx *context) {
	socket := goutils.SocketSettings()
	socket.Bind = ctx.binding
	onReceive := &receiver{}
	goutils.WriteInfo("ready to receive socket information")
	goutils.SocketReceiveOnly(socket, onReceive)
}
