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
	dumpKey   = "dump"
	fKey      = "field"
	field0Key = fKey + "0"
	field1Key = fKey + "1"
	field2Key = fKey + "2"
	field3Key = fKey + "3"
	field4Key = fKey + "4"
	field5Key = fKey + "5"
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

func isRaw(e *Entry) bool {
	return e.Type == notJSON
}

func isArray(e *Entry) bool {
	return e.Type == arrayJSON
}

func isObject(e *Entry) bool {
	return e.Type == objJSON
}

func isNotRaw(e *Entry) bool {
	return isObject(e) || isArray(e)
}

func isTag(e *Entry) bool {
	if isRaw(e) && len(e.Raw) == 4 {
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
		p.name = fmt.Sprintf("%s%d", fKey, idx)
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

func handleAll(entries map[string]*Entry, h entityHandler) map[string]*Entry {
	return h.handle(len(entries), entries)
}

type handlerSettings struct {
	allowEvent bool
	allowDump  bool
	enabled    bool
}

func handleEntries(entries map[string]*Entry, settings *handlerSettings) map[string]*Entry {
	if len(entries) == 0 {
		return entries
	}
	var handler entityHandler
	handler = &defaultHandler{}
	first, ok := entries[field0Key]
	if ok {
		if settings.allowEvent {
			if isRaw(first) && first.Raw == "event" {
				handler = &eventHandler{}
			}
		}
	}
	r := make(map[string]*Entry)
	for _, v := range handleAll(entries, handler) {
		r[v.name] = v
	}
	return r
}

type entityHandler interface {
	handle(int, map[string]*Entry) map[string]*Entry
}

type defaultHandler struct {
	entityHandler
}

// default handler is a noop, we don't know what to do with this entity
func (h *defaultHandler) handle(count int, entries map[string]*Entry) map[string]*Entry {
	return entries
}

// handles anything marked as an event
type eventHandler struct {
	entityHandler
}

type entityCheck func(e *Entry) bool

func rewriteName(name, field string, check entityCheck, entries map[string]*Entry) bool {
	v, ok := entries[field]
	if !ok {
		return false
	}
	ok = check(v)
	if ok {
		v.name = name
	}
	return ok
}

func set(e *Entry) bool {
	return true
}

func (h *eventHandler) handle(count int, entries map[string]*Entry) map[string]*Entry {
	rewriteName("event", field0Key, set, entries)
	if rewriteName("tag", field1Key, isTag, entries) {
		if rewriteName("playerid", field2Key, isRaw, entries) {
			if rewriteName("type", field3Key, isRaw, entries) {
				if rewriteName("data", field4Key, isNotRaw, entries) {
					rewriteName("simtime", field5Key, isRaw, entries)
				}
			}
		}
	}
	return entries
}
