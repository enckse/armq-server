package common

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// Startup is a common way to setup command-line application in the armq-* portfolio
func Startup(vers string) *Configuration {
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
	Info(vers)
	return c
}

type (
	// TypeConv is an indicator ot type conversion
	TypeConv int
	// OpType is an operator type
	OpType int

	// Entry represents a data entry
	Entry struct {
		// Corresponding type (of data to query from other fields)
		Type string `json:"jsontype"`
		// Represents a raw field of data
		Raw string `json:"raw,omitempty"`
		// Represents an array object
		Array []json.RawMessage `json:"array,omitempty"`
		// Represents a map (object)
		Object map[string]json.RawMessage `json:"object,omitempty"`
		Name   string                     `json:"-"`
	}

	// Configuration for the server
	Configuration struct {
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
)

// HandleFields indicates if the handlers support field handling
func (c *Configuration) HandleFields() bool {
	return c.API.Handlers.Event || c.API.Handlers.Start || c.API.Handlers.Player || c.API.Handlers.Replay
}

// Info is for informational messages
func Info(message string) {
	fmt.Println(message)
}

// Errored is for error messaging
func Errored(message string, err error) {
	Info(fmt.Sprintf("ERROR -> %s (%v)", message, err))
}
