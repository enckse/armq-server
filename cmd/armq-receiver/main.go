package main

import (
	"voidedtech.com/armq-server/internal/receiver"
)

var vers = "master"

func main() {
	receiver.Run(vers)
}
