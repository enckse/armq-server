package main

import (
	"flag"
	"sync"

	"github.com/epiphyte/goutils"
)

var (
	gcLock   = &sync.Mutex{}
	readLock = &sync.Mutex{}
	memcache = make(map[string]struct{})
	objcache = []*object{}
	gc       = []string{}
)

const (
	fileMode = "file"
	sockMode = "socket"
)

type object struct {
	id   string
	data []byte
	gc   bool
}

type reader struct {
}

func collect() []string {
	gcLock.Lock()
	defer gcLock.Unlock()
	res := []string{}
	for _, v := range gc {
		res = append(res, v)
	}
	gc = []string{}
	return res
}

func garbage(obj *object) {
	if obj.gc {
		gcLock.Lock()
		defer gcLock.Unlock()
		gc = append(gc, obj.id)
	}
}

func queue(id string, data []byte, gc bool) {
	readLock.Lock()
	defer readLock.Unlock()
	if _, ok := memcache[id]; ok {
		goutils.WriteWarn("system error? trying to re-add id", id)
		return
	}
	memcache[id] = struct{}{}
	objcache = append(objcache, &object{id: id, data: data, gc: gc})
}

func next() (*object, bool) {
	readLock.Lock()
	defer readLock.Unlock()
	if len(objcache) == 0 {
		return nil, false
	}
	obj := objcache[0]
	objcache = objcache[1:]
	return obj, true
}

func main() {
	bind := flag.String("bind", "127.0.0.1:5000", "binding address")
	mode := flag.String("mode", fileMode, "receiving mode")
	debug := flag.Bool("debug", false, "enable debugging")
	flag.Parse()
	opts := goutils.NewLogOptions()
	opts.Debug = *debug
	goutils.ConfigureLogging(opts)
	switch *mode {
	case sockMode:
		go socketReceiver(*bind, *debug)
	case fileMode:
	default:
		goutils.Fatal("unknown mode", nil)
	}
}
