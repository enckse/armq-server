package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"strings"
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
	fileMode   = "file"
	sockMode   = "socket"
	repeatMode = "repeat"
	delimiter  = "`"
)

type context struct {
	directory  string
	binding    string
	debug      bool
	start      time.Time
	timeFormat string
	repeater   bool
}

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

type Datum struct {
	Id        string `json:"id"`
	Timestamp string `json:"ts"`
	Version   string `json:"vers"`
	Raw       string `json:"raw"`
	File      string `json:"file"`
	Date      string `json:"datetime"`
}

func writerWorker(id, count int, obj *object, ctx *context) bool {
	datum := &Datum{}
	datum.Raw = string(obj.data)
	parts := strings.Split(datum.Raw, delimiter)
	datum.Timestamp = parts[0]
	i, e := strconv.ParseInt(datum.Timestamp, 10, 64)
	if e != nil {
		goutils.WriteWarn("unable to parse timestamp (not critical)", obj.id)
		goutils.WriteError("parse error was", e)
	}
	datum.Date = time.Unix(i/1000, 0).Format("2006-01-02T15:04:05")
	datum.Version = parts[1]
	datum.File = obj.id
	datum.Id = fmt.Sprintf("%s.%s.%d", ctx.timeFormat, datum.Timestamp, count)
	j, e := json.Marshal(datum)
	if e != nil {
		goutils.WriteWarn("unable to handle file", obj.id)
		goutils.WriteError("unable to read object to json", e)
		return false
	}
	goutils.WriteDebug(string(j))
	return true
}

func repeaterWorker(socket *goutils.SocketSetup, obj *object) bool {
	err := goutils.SocketSendOnly(socket, obj.data)
	if err != nil {
		goutils.WriteError("unable to send data over socket", err)
		return false
	}
	return true
}

func createWorker(id int, ctx *context) {
	count := 0
	var socket *goutils.SocketSetup
	if ctx.repeater {
		socket = goutils.SocketSettings()
		socket.Bind = ctx.binding
	}
	for {
		obj, ok := next()
		if ok {
			goutils.WriteDebug(fmt.Sprintf("%d -> %s", id, obj.id))
			if ctx.repeater {
				ok = repeaterWorker(socket, obj)
			} else {
				if writerWorker(id, count, obj, ctx) {
					count += 1
				} else {
					ok = false
				}
			}
			if ok {
				garbage(obj)
			}
		}
		if !ok {
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func main() {
	bind := flag.String("bind", "127.0.0.1:5000", "binding address")
	mode := flag.String("mode", fileMode, "receiving mode")
	dir := flag.String("directory", "/dev/shm/armq/", "location to scan for files to read")
	debug := flag.Bool("debug", false, "enable debugging")
	workers := flag.Int("workers", 4, "worker routines")
	now := time.Now()
	op := *mode
	flag.Parse()
	ctx := &context{}
	ctx.directory = *dir
	ctx.debug = *debug
	ctx.binding = *bind
	ctx.start = now
	ctx.repeater = op == repeatMode
	ctx.timeFormat = now.Format("2006-01-02T15-04-05")
	opts := goutils.NewLogOptions()
	opts.Debug = ctx.debug
	goutils.ConfigureLogging(opts)
	goutils.WriteInfo("starting", vers, op)
	switch op {
	case sockMode:
		go socketReceiver(ctx)
	case fileMode, repeatMode:
		go fileReceive(ctx)
	default:
		goutils.Fatal("unknown mode", nil)
	}
	worker := *workers
	if ctx.repeater {
		if worker != 1 {
			goutils.WriteWarn("setting workers back to 1 for repeater")
			worker = 1
		}
	}
	i := 0
	for i < worker {
		go createWorker(i, ctx)
		i += 1
	}
	for {
		time.Sleep(1)
	}
}
