package main

import (
	"voidedtech.com/armq-server/internal"
)

var vers = "master"

func main() {
	internal.RunReceiver(vers)
}
