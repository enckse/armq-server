package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"voidedtech.com/armq-server/internal/common"
)

const (
	eventType    = "event"
	startType    = "start"
	replayType   = "replay"
	playerType   = "player"
	playerIDType = "playerid"
	emptyJSON    = "empty"
	field0Key    = common.FKey + "0"
	field1Key    = common.FKey + "1"
	field2Key    = common.FKey + "2"
	field3Key    = common.FKey + "3"
	field4Key    = common.FKey + "4"
	field5Key    = common.FKey + "5"
)

func isEmpty(e *common.Entry) bool {
	return len(e.Raw) == 0 && len(e.Array) == 0 && len(e.Object) == 0
}

func isRaw(e *common.Entry) bool {
	return e.Type == common.NotJSON
}

func isArray(e *common.Entry) bool {
	return e.Type == common.ArrayJSON
}

func isObject(e *common.Entry) bool {
	return e.Type == common.ObjJSON
}

func isNotRaw(e *common.Entry) bool {
	return isObject(e) || isArray(e)
}

func isTag(e *common.Entry) bool {
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

func loadFile(path string, h *common.Configuration) (map[string]json.RawMessage, []byte) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		common.Info(fmt.Sprintf("error reading file: %s", path))
		common.Errored("unable to read file", err)
		return nil, nil
	}
	var obj map[string]json.RawMessage
	err = json.Unmarshal(b, &obj)
	if err != nil {
		common.Info(fmt.Sprintf("unable to marshal object: %s", path))
		common.Errored("unable to parse json", err)
		return nil, nil
	}
	if h.API.Handlers.Enable {
		if !h.API.Handlers.Dump {
			_, ok := obj[common.DumpKey]
			if ok {
				delete(obj, common.DumpKey)
			}
		}
		v, ok := obj[common.FieldKey]
		if ok {
			var fields map[string]*common.Entry
			err = json.Unmarshal(v, &fields)
			if err == nil {
				rewrite := handleEntries(fields, h)
				r, err := json.Marshal(rewrite)
				if err == nil {
					obj[common.FieldKey] = r
					b, _ = json.Marshal(obj)
				}
			}
		}
	}
	return obj, b
}

func handleAll(entries map[string]*common.Entry, h entityHandler) map[string]*common.Entry {
	return h.handle(len(entries), entries)
}

func handleEntries(entries map[string]*common.Entry, settings *common.Configuration) map[string]*common.Entry {
	if len(entries) == 0 {
		return entries
	}
	var handler entityHandler
	handler = &defaultHandler{}
	first, ok := entries[field0Key]
	if ok {
		firstField := ""
		if settings.HandleFields() {
			if isRaw(first) {
				firstField = first.Raw
			}
		}
		if settings.API.Handlers.Event && firstField == eventType {
			handler = &eventHandler{}
		}
		if settings.API.Handlers.Start && firstField == startType {
			handler = &startHandler{}
		}
		if settings.API.Handlers.Replay && firstField == replayType {
			handler = &replayHandler{}
		}
		if settings.API.Handlers.Player && firstField == playerType {
			handler = &playerHandler{}
		}
	}
	r := make(map[string]*common.Entry)
	for k, v := range handleAll(entries, handler) {
		n := strings.TrimSpace(v.Name)
		if len(n) == 0 {
			n = k
		}
		if settings.API.Handlers.Empty {
			if isEmpty(v) {
				v.Type = emptyJSON
			}
		}
		r[n] = v
	}
	return r
}

type entityHandler interface {
	handle(int, map[string]*common.Entry) map[string]*common.Entry
}

type defaultHandler struct {
	entityHandler
}

// default handler is a noop, we don't know what to do with this entity
func (h *defaultHandler) handle(count int, entries map[string]*common.Entry) map[string]*common.Entry {
	return entries
}

// handles anything marked as an event
type eventHandler struct {
	entityHandler
}

type entityCheck func(e *common.Entry) bool

func rewriteName(name, field string, check entityCheck, entries map[string]*common.Entry) bool {
	v, ok := entries[field]
	if !ok {
		return false
	}
	ok = check(v)
	if ok {
		v.Name = name
	}
	return ok
}

func set(e *common.Entry) bool {
	return true
}

func (h *eventHandler) handle(count int, entries map[string]*common.Entry) map[string]*common.Entry {
	rewriteName(eventType, field0Key, set, entries)
	if rewriteName(common.TagKey, field1Key, isTag, entries) {
		if rewriteName(playerIDType, field2Key, isRaw, entries) {
			if rewriteName("type", field3Key, isRaw, entries) {
				if rewriteName("data", field4Key, isNotRaw, entries) {
					rewriteName("simtime", field5Key, isRaw, entries)
				}
			}
		}
	}
	return entries
}

// indicates a 'start'
type startHandler struct {
	entityHandler
}

func (h *startHandler) handle(count int, entries map[string]*common.Entry) map[string]*common.Entry {
	rewriteName(startType, field0Key, set, entries)
	return entries
}

// indicates a 'player'
type playerHandler struct {
	entityHandler
}

func (h *playerHandler) handle(count int, entries map[string]*common.Entry) map[string]*common.Entry {
	rewriteName(playerType, field0Key, set, entries)
	if rewriteName(playerIDType, field1Key, isRaw, entries) {
		rewriteName("name", field2Key, isRaw, entries)
	}
	return entries
}

// indicates a 'replay'
type replayHandler struct {
	entityHandler
}

func (h *replayHandler) handle(count int, entries map[string]*common.Entry) map[string]*common.Entry {
	rewriteName(replayType, field0Key, set, entries)
	if rewriteName("mission", field1Key, isTag, entries) {
		if rewriteName("world", field2Key, isRaw, entries) {
			if rewriteName("daytime", field3Key, isRaw, entries) {
				rewriteName("version", field4Key, isNotRaw, entries)
			}
		}
	}
	return entries
}
