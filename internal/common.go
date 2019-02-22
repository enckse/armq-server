package internal

import (
	"encoding/json"
	"flag"

	"voidedtech.com/goutils/logger"
	"voidedtech.com/goutils/preyaml"
)

var (
	vers        = "master"
	emptyObject = []byte("{}")
)

func startup() *Configuration {
	conf := flag.String("config", "/etc/armq.yaml", "config file")
	flag.Parse()
	d := &preyaml.Directives{}
	c := &Configuration{}
	err := preyaml.UnmarshalFile(*conf, d, c)
	if err != nil {
		logger.Fatal("unable to parse config", err)
	}
	logger.WriteInfo("starting", vers)
	opts := logger.NewLogOptions()
	opts.Debug = c.Global.Debug
	logger.ConfigureLogging(opts)
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

type Configuration struct {
	Global struct {
		Bind    string
		Debug   bool
		Workers int
		Output  string
		Dump    bool
	}
	Files struct {
		Directory string
		Gc        int
		After     int
		Sleep     int
	}
	Api struct {
		Bind      string
		Limit     int
		Top       int
		StartScan int
		EndScan   int
		Handlers  struct {
			Enable bool
			Dump   bool
			Event  bool
			Empty  bool
			Start  bool
			Replay bool
			Player bool
		}
	}
}
