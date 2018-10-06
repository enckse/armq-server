package main

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/epiphyte/goutils"
)

func isEmpty(e *Entry) bool {
	return len(e.Raw) == 0 && len(e.Array) == 0 && len(e.Object) == 0
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

func loadFile(path string, h *handlerSettings) (map[string]json.RawMessage, []byte) {
	goutils.WriteDebug("reading", path)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		goutils.WriteWarn("error reading file", path)
		goutils.WriteError("unable to read file", err)
		return nil, nil
	}
	var obj map[string]json.RawMessage
	err = json.Unmarshal(b, &obj)
	if err != nil {
		goutils.WriteWarn("unable to marshal object", path)
		goutils.WriteError("unable to parse json", err)
		return nil, nil
	}
	if h.enabled {
		if !h.allowDump {
			_, ok := obj[dumpKey]
			if ok {
				delete(obj, dumpKey)
			}
		}
		v, ok := obj[fieldKey]
		if ok {
			var fields map[string]*Entry
			err = json.Unmarshal(v, &fields)
			if err == nil {
				rewrite := handleEntries(fields, h)
				r, err := json.Marshal(rewrite)
				if err == nil {
					obj[fieldKey] = r
					b, _ = json.Marshal(obj)
				}
			}
		}
	}
	return obj, b
}

func handleAll(entries map[string]*Entry, h entityHandler) map[string]*Entry {
	return h.handle(len(entries), entries)
}

type handlerSettings struct {
	allowEvent bool
	allowDump  bool
	allowEmpty bool
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
	for k, v := range handleAll(entries, handler) {
		n := strings.TrimSpace(v.name)
		if len(n) == 0 {
			n = k
		}
		if settings.allowEmpty {
			if isEmpty(v) {
				v.Type = emptyJSON
			}
		}
		r[n] = v
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
