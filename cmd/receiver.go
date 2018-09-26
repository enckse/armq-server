package main

import (
	"flag"

	"github.com/epiphyte/goutils"
)

const (
	fileMode = "file"
	sockMode = "socket"
)

func main() {
	bind := flag.String("bind", "127.0.0.1:5000", "binding address")
	mode := flag.String("mode", fileMode, "receiving mode")
	debug := flag.Bool("debug", false, "enable debugging")
	flag.Parse()
	opts := goutils.NewLogOptions()
	opts.Debug = *debug
	goutils.ConfigureLogging(opts)
	switch *mode {
	case sockMode:
		socketReceiver(*bind, *debug)
	case fileMode:
	default:
		goutils.Fatal("unknown mode", nil)
	}
}
