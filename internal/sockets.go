package internal

import (
	"voidedtech.com/goutils/logger"
	"voidedtech.com/goutils/sockets"
)

type receiver struct {
	sockets.SocketReceive
}

func (r *receiver) Consume(d []byte) {
	logger.WriteInfo(string(d))
	queue("", d, false)
}

func socketReceiver(ctx *rcvContext) {
	logger.WriteInfo("socket mode enabled")
	socket := sockets.SocketSettings()
	socket.Bind = ctx.binding
	onReceive := &receiver{}
	logger.WriteInfo("ready to receive socket information")
	sockets.SocketReceiveOnly(socket, onReceive)
}
