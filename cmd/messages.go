package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	//	"strings"

	"github.com/epiphyte/goutils"
)

const (
	notJSON   = "raw"
	objJSON   = "obj"
	arrayJSON = "array"
	skipJSON  = "na"
	delimiter = "`"
	emptyJSON = "{\"jsontype\": \"" + skipJSON + "\"}"
)

var (
	emptyObject = []byte(emptyJSON)
)

type Entry struct {
	Type   string                     `json:"jsontype"`
	Raw    string                     `json:"raw,omitempty"`
	Array  []json.RawMessage          `json:"array,omitempty"`
	Object map[string]json.RawMessage `json:"object,omitempty"`
	name   string
}

func detectJSON(segment []string) string {
	if len(segment) == 0 {
		return ""
	}
	entries := []*Entry{}
	for idx, section := range segment {
		p := &Entry{}
		p.Type = notJSON
		p.Raw = section
		var arr []json.RawMessage
		bytes := []byte(section)
		if json.Unmarshal(bytes, &arr) == nil {
			p.Array = arr
			p.Type = arrayJSON
		} else {
			var obj map[string]json.RawMessage
			if json.Unmarshal(bytes, &obj) == nil {
				p.Type = objJSON
				p.Object = obj
			}
		}
		p.name = fmt.Sprintf("field%d", idx)
		entries = append(entries, p)
	}
	entries = handleEntries(entries)
	var buffer bytes.Buffer
	for idx, e := range handleEntries(entries) {
		if idx > 0 {
			buffer.WriteString(",")
		}
		entry := &Entry{Type: e.Type}
		switch e.Type {
		case notJSON:
			entry.Raw = e.Raw
		case arrayJSON:
			entry.Array = e.Array
		case objJSON:
			entry.Object = e.Object
		}
		j, err := json.Marshal(entry)
		if err != nil {
			goutils.WriteError("unable to marshal raw object", err)
			j = emptyObject
		}
		buffer.WriteString(fmt.Sprintf("\"%s\": ", e.name))
		buffer.Write(j)
	}
	return fmt.Sprintf("{%s}", buffer.String())
}

// This is where we can rewrite field designations depending on our inputs
// from generic fieldN to a valid, useful name
func handleEntries(entries []*Entry) []*Entry {
	return entries
}
