package messages

import (
	"strings"

	"voidedtech.com/armq-server/internal"
)

const (
	eventType    = "event"
	startType    = "start"
	replayType   = "replay"
	playerType   = "player"
	playerIDType = "playerid"
	emptyJSON    = "empty"
	field0Key    = internal.FKey + "0"
	field1Key    = internal.FKey + "1"
	field2Key    = internal.FKey + "2"
	field3Key    = internal.FKey + "3"
	field4Key    = internal.FKey + "4"
	field5Key    = internal.FKey + "5"
)

func isEmpty(e *internal.Entry) bool {
	return len(e.Raw) == 0 && len(e.Array) == 0 && len(e.Object) == 0
}

func isRaw(e *internal.Entry) bool {
	return e.Type == internal.NotJSON
}

func isArray(e *internal.Entry) bool {
	return e.Type == internal.ArrayJSON
}

func isObject(e *internal.Entry) bool {
	return e.Type == internal.ObjJSON
}

func isNotRaw(e *internal.Entry) bool {
	return isObject(e) || isArray(e)
}

func isTag(e *internal.Entry) bool {
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

func handleAll(entries map[string]*internal.Entry, h entityHandler) map[string]*internal.Entry {
	return h.handle(len(entries), entries)
}

// HandleEntries is responsible for taking a set of input entries and massaging them into useful data
func HandleEntries(entries map[string]*internal.Entry, settings *internal.Configuration) map[string]*internal.Entry {
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
	r := make(map[string]*internal.Entry)
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

type (
	// indicates a 'start'
	startHandler struct {
		entityHandler
	}

	// indicates a 'player'
	playerHandler struct {
		entityHandler
	}

	// indicates a 'replay'
	replayHandler struct {
		entityHandler
	}

	entityHandler interface {
		handle(int, map[string]*internal.Entry) map[string]*internal.Entry
	}

	defaultHandler struct {
		entityHandler
	}

	// handles anything marked as an event
	eventHandler struct {
		entityHandler
	}

	entityCheck func(e *internal.Entry) bool
)

// default handler is a noop, we don't know what to do with this entity
func (h *defaultHandler) handle(count int, entries map[string]*internal.Entry) map[string]*internal.Entry {
	return entries
}

func rewriteName(name, field string, check entityCheck, entries map[string]*internal.Entry) bool {
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

func set(e *internal.Entry) bool {
	return true
}

func (h *eventHandler) handle(count int, entries map[string]*internal.Entry) map[string]*internal.Entry {
	rewriteName(eventType, field0Key, set, entries)
	if rewriteName(internal.TagKey, field1Key, isTag, entries) {
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

func (h *startHandler) handle(count int, entries map[string]*internal.Entry) map[string]*internal.Entry {
	rewriteName(startType, field0Key, set, entries)
	return entries
}

func (h *playerHandler) handle(count int, entries map[string]*internal.Entry) map[string]*internal.Entry {
	rewriteName(playerType, field0Key, set, entries)
	if rewriteName(playerIDType, field1Key, isRaw, entries) {
		rewriteName("name", field2Key, isRaw, entries)
	}
	return entries
}

func (h *replayHandler) handle(count int, entries map[string]*internal.Entry) map[string]*internal.Entry {
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
