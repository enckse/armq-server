package main

import (
	"flag"

	"github.com/epiphyte/goutils"
)

var (
	vers    = "master"
	dataDir = "/var/lib/armq/"
	outKey  = "output"
)

func startup() *goutils.Config {
	conf := flag.String("config", "/etc/armq.conf", "config file")
	flag.Parse()
	c, e := goutils.LoadConfigDefaults(*conf)
	if e != nil {
		goutils.Fatal("failed to start", e)
	}
	goutils.WriteInfo("starting", vers)
	opts := goutils.NewLogOptions()
	opts.Debug = c.GetTrue("debug")
	goutils.ConfigureLogging(opts)
	return c
}
