package internal

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

var (
	vers        = "master"
	emptyObject = []byte("{}")
)

func startup() *Configuration {
	conf := flag.String("config", "/etc/armq.conf", "config file")
	flag.Parse()
	c := &Configuration{}
	b, err := ioutil.ReadFile(*conf)
	if err != nil {
		panic(fmt.Sprintf("unable to read config %v", err))
	}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		panic(fmt.Sprintf("unable to parse config %v", err))
	}
	info(vers)
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

// Entry represents a data entry
type Entry struct {
	// Corresponding type (of data to query from other fields)
	Type string `json:"jsontype"`
	// Represents a raw field of data
	Raw string `json:"raw,omitempty"`
	// Represents an array object
	Array []json.RawMessage `json:"array,omitempty"`
	// Represents a map (object)
	Object map[string]json.RawMessage `json:"object,omitempty"`
	name   string
}

// Configuration for the server
type Configuration struct {
	Global struct {
		Bind    string
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
	API struct {
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

func info(message string) {
	fmt.Println(message)
}

func errored(message string, err error) {
	info(fmt.Sprintf("ERROR -> %s (%v)", message, err))
}
