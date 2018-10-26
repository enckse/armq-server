package main

import (
	"encoding/json"
	"flag"

	"github.com/epiphyte/goutils"
)

var (
	vers        = "master"
	dataDir     = "/var/lib/armq/"
	outKey      = "output"
	emptyObject = []byte("{}")
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

type typeConv int
type opType int

const (
	maxOp = 5
	minOp = -1

	notJSON          = "raw"
	objJSON          = "object"
	arrayJSON        = "array"
	emptyJSON        = "empty"
	delimiter        = "`"
	fieldKey         = "fields"
	dumpKey          = "dump"
	fKey             = "field"
	tsKey            = "ts"
	idKey            = "id"
	tagKey           = "tag"
	dtKey            = "dt"
	field0Key        = fKey + "0"
	field1Key        = fKey + "1"
	field2Key        = fKey + "2"
	field3Key        = fKey + "3"
	field4Key        = fKey + "4"
	field5Key        = fKey + "5"
	lessThan  opType = 0
	equals    opType = 1
	lessTE    opType = 2
	greatThan opType = 3
	greatTE   opType = 4
	nEquals   opType = maxOp
	invalidOp opType = minOp
)

type Entry struct {
	Type   string                     `json:"jsontype"`
	Raw    string                     `json:"raw,omitempty"`
	Array  []json.RawMessage          `json:"array,omitempty"`
	Object map[string]json.RawMessage `json:"object,omitempty"`
	name   string
}
