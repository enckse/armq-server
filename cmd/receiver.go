package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
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
)

const (
	fileMode      = "file"
	sockMode      = "socket"
	repeatMode    = "repeat"
	sleepCycleMin = 90
	sleepCycleMax = 108
)

type context struct {
	binding    string
	start      time.Time
	timeFormat string
	repeater   bool
	output     string
	dump       bool
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
	requeue(&object{id: id, data: data, gc: gc})
}

func requeue(obj *object) {
	readLock.Lock()
	defer readLock.Unlock()
	objcache = append(objcache, obj)
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
	Id        string
	Timestamp int64
	Version   string
	File      string
	Date      string
}

func (d *Datum) toJSON() string {
	return fmt.Sprintf("\"id\": \"%s\", \"%s\": %d, \"vers\": \"%s\", \"file\": \"%s\", \"dt\": \"%s\"", d.Id, tsJSON, d.Timestamp, d.Version, d.File, d.Date)
}

func writerWorker(id, count int, outdir string, obj *object, ctx *context) bool {
	dump := &Entry{Raw: string(obj.data), Type: notJSON}
	datum := &Datum{}
	parts := strings.Split(dump.Raw, delimiter)
	ts := parts[0]
	i, e := strconv.ParseInt(ts, 10, 64)
	if e != nil {
		goutils.WriteWarn("unable to parse timestamp (not critical)", obj.id)
		goutils.WriteError("parse error was", e)
		i = -1
	}
	datum.Timestamp = i
	datum.Date = time.Unix(i/1000, 0).Format("2006-01-02T15:04:05")
	datum.Version = parts[1]
	datum.File = obj.id
	datum.Id = fmt.Sprintf("%s.%d.%d.%d", ctx.timeFormat, datum.Timestamp, id, count)
	fields := detectJSON(parts[2:])
	if fields == "" {
		fields = "{}"
	}
	j := emptyObject
	if ctx.dump {
		j, e = json.Marshal(dump)
		if e != nil {
			goutils.WriteWarn("unable to handle file", obj.id)
			goutils.WriteError("unable to read object to json", e)
			return false
		}
	}
	j = []byte(fmt.Sprintf("{%s, \"dump\": %s, \"%s\": %s}", datum.toJSON(), j, fieldKey, fields))
	goutils.WriteDebug(string(j))
	p := filepath.Join(outdir, datum.Id)
	err := ioutil.WriteFile(p, j, 0644)
	if err != nil {
		goutils.WriteWarn("error saving results", p)
		goutils.WriteError("unable to save file", err)
		return false
	}
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

func (c *context) resetWorker() (int, string) {
	now := time.Now().Format("2006-01-02")
	p := filepath.Join(c.output, now)
	goutils.RunBashCommand(fmt.Sprintf("mkdir -p %s", p))
	return 0, p
}

func createWorker(id int, ctx *context) {
	count, outdir := ctx.resetWorker()
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
				if writerWorker(id, count, outdir, obj, ctx) {
					count += 1
				} else {
					ok = false
				}
			}
			if ok {
				garbage(obj)
			} else {
				requeue(obj)
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
				count, outdir = ctx.resetWorker()
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
	run(startup())
}

func run(config *goutils.Config) {
	now := time.Now()
	op := config.GetStringOrDefault("mode", fileMode)
	ctx := &context{}
	ctx.binding = config.GetStringOrDefault("bind", "127.0.0.1:5000")
	ctx.start = now
	ctx.repeater = op == repeatMode
	ctx.timeFormat = now.Format("2006-01-02T15-04-05")
	ctx.output = config.GetStringOrDefault("output", dataDir)
	ctx.dump = config.GetTrue("dump")
	section := fmt.Sprintf("[%s]", op)
	switch op {
	case sockMode:
		go socketReceiver(ctx)
	case fileMode, repeatMode:
		go fileReceive(config.GetSection(section))
	default:
		goutils.Fatal("unknown mode", nil)
	}
	worker := config.GetIntOrDefaultOnly("workers", 4)
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
