package main

import (
	"voidedtech.com/armq-server/internal/api"
)

var vers = "master"

func main() {
	api.Run(vers)
}
