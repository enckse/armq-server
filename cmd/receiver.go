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
	fileMode      = "file"
	sockMode      = "socket"
	repeatMode    = "repeat"
	delimiter     = "`"
	sleepCycleMin = 90
	sleepCycleMax = 108
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
	lastWorked := 0
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
		if ok {
			lastWorked = 0
		} else {
			cooldown := 1
			switch {
			case lastWorked < sleepCycleMin:
				cooldown = 1
			case lastWorked >= sleepCycleMin && lastWorked < sleepCycleMax:
				cooldown = 5
			case id > 0 && lastWorked >= sleepCycleMax:
				// initial worker can never go this slow
				cooldown = 30
			}
			if lastWorked < sleepCycleMax {
				lastWorked += 1
			}
			sleepFor := time.Duration(cooldown) * time.Second
			time.Sleep(sleepFor)
		}
	}
}

func main() {
	conf := flag.String("config", "/etc/armq.conf", "configuration file")
	flag.Parse()
	c, e := goutils.LoadConfigDefaults(*conf)
	if e != nil {
		goutils.WriteError("unable to read config", e)
		return
	}
	run(c)
}

func run(config *goutils.Config) {
	now := time.Now()
	op := config.GetStringOrDefault("mode", fileMode)
	ctx := &context{}
	ctx.directory = config.GetStringOrDefault("directory", "/dev/shm/armq/")
	ctx.debug = config.GetTrue("debug")
	ctx.binding = config.GetStringOrDefault("bind", "127.0.0.1:5000")
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
	worker, err := config.GetIntOrDefault("workers", 4)
	if err != nil {
		goutils.WriteError("invalid worker settings", err)
		worker = 4
	}
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
