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
	objJSON   = "object"
	arrayJSON = "array"
	delimiter = "`"
	fieldKey  = "fields"
)

var (
	emptyObject = []byte("{}")
)

type Entry struct {
	Type   string                     `json:"jsontype"`
	Raw    string                     `json:"raw,omitempty"`
	Array  []json.RawMessage          `json:"array,omitempty"`
	Object map[string]json.RawMessage `json:"object,omitempty"`
	name   string
}

func (e *Entry) isRaw() bool {
	return e.Type == notJSON
}

func (e *Entry) isArray() bool {
	return e.Type == arrayJSON
}

func (e *Entry) isObject() bool {
	return e.Type == objJSON
}

func (e *Entry) isNotRaw() bool {
	return e.isObject() || e.isArray()
}

func (e *Entry) isTag() bool {
	if e.isRaw() && len(e.Raw) == 4 {
		for _, r := range e.Raw {
			if r >= 'a' && r <= 'z' {
				continue
			}
			return false
		}
		return true
	}
	return false
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
	var buffer bytes.Buffer
	for idx, e := range entries {
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

func handleAll(entries []*Entry, h entityHandler) []*Entry {
	return h.handle(len(entries), entries)
}

type handlerSettings struct {
	allowEvent bool
}

func handleEntries(entries []*Entry, settings *handlerSettings) []*Entry {
	if len(entries) == 0 {
		return entries
	}
	var handler entityHandler
	handler = &defaultHandler{}
	first := entries[0]
	if settings.allowEvent {
		if first.isRaw() && first.Raw == "event" {
			handler = &eventHandler{}
		}
	}
	return handleAll(entries, handler)
}

type entityHandler interface {
	handle(int, []*Entry) []*Entry
}

type defaultHandler struct {
	entityHandler
}

// default handler is a noop, we don't know what to do with this entity
func (h *defaultHandler) handle(count int, entries []*Entry) []*Entry {
	return entries
}

// handles anything marked as an event
type eventHandler struct {
	entityHandler
}

func (h *eventHandler) handle(count int, entries []*Entry) []*Entry {
	entries[0].name = "event"
	if count > 1 && entries[1].isTag() {
		entries[1].name = "tag"
		if count > 2 && entries[2].isRaw() {
			entries[2].name = "playerid"
			if count > 3 && entries[3].isRaw() {
				entries[3].name = "type"
				if count > 4 && entries[4].isNotRaw() {
					entries[4].name = "data"
					if count > 5 && entries[5].isRaw() {
						entries[5].name = "simtime"
					}
				}
			}
		}
	}
	return entries
}
