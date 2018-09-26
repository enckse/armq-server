package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/epiphyte/goutils"
)

var (
	gcLock   = &sync.Mutex{}
	readLock = &sync.Mutex{}
	objcache = []*object{}
	gc       = []string{}
	vers     = "master"
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

func createWorker(id int) {
	count := 0
	for {
		obj, ok := next()
		if ok {
			goutils.WriteInfo(fmt.Sprintf("%d -> %s", id, obj.id))
			garbage(obj)
		}
		count += 1
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	bind := flag.String("bind", "127.0.0.1:5000", "binding address")
	mode := flag.String("mode", fileMode, "receiving mode")
	dir := flag.String("directory", "/dev/shm/armq/", "location to scan for files to read")
	debug := flag.Bool("debug", false, "enable debugging")
	workers := flag.Int("workers", 4, "worker routines")
	flag.Parse()
	opts := goutils.NewLogOptions()
	opts.Debug = *debug
	goutils.ConfigureLogging(opts)
	goutils.WriteInfo("starting", vers)
	switch *mode {
	case sockMode:
		go socketReceiver(*bind)
	case fileMode:
		go fileReceive(*dir)
	default:
		goutils.Fatal("unknown mode", nil)
	}
	i := 0
	for i < *workers {
		go createWorker(i)
		i += 1
	}
	for {
		time.Sleep(1)
	}
}
