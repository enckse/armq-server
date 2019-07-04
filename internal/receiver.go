package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	gcLock   = &sync.Mutex{}
	readLock = &sync.Mutex{}
	objcache = []*object{}
	gc       = []string{}
	lock     = &sync.Mutex{}
	cache    = make(map[string]struct{})
)

const (
	fileMode      = "file"
	sleepCycleMin = 90
	sleepCycleMax = 108
)

type rcvContext struct {
	timeFormat string
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
	return fmt.Sprintf("\"%s\": \"%s\", \"%s\": %d, \"vers\": \"%s\", \"file\": \"%s\", \"%s\": \"%s\"", idKey, d.Id, tsKey, d.Timestamp, d.Version, d.File, dtKey, d.Date)
}

func writerWorker(id, count int, outdir string, obj *object, ctx *rcvContext) bool {
	dump := &Entry{Raw: string(obj.data), Type: notJSON}
	datum := &Datum{}
	parts := strings.Split(dump.Raw, delimiter)
	ts := parts[0]
	i, e := strconv.ParseInt(ts, 10, 64)
	if e != nil {
		info(fmt.Sprintf("unable to parse timestamp (not critical): %s", obj.id))
		errored("parse error was", e)
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
			info(fmt.Sprintf("unable to handle file %s", obj.id))
			errored("unable to read object to json", e)
			return false
		}
	}
	j = []byte(fmt.Sprintf("{%s, \"%s\": %s, \"%s\": %s}", datum.toJSON(), dumpKey, j, fieldKey, fields))
	p := filepath.Join(outdir, datum.Id)
	err := ioutil.WriteFile(p, j, 0644)
	if err != nil {
		info(fmt.Sprintf("error saving results: %s", p))
		errored("unable to save file", err)
		return false
	}
	return true
}

func (c *rcvContext) resetWorker() (int, string) {
	now := time.Now().Format("2006-01-02")
	p := filepath.Join(c.output, now)
	if !pathExists(p) {
		err := os.MkdirAll(p, 0755)
		if err != nil {
			info(fmt.Sprintf("error reseting path: %s", p))
			errored("error for path reset", err)
		}
	}
	return 0, p
}

func createWorker(id int, ctx *rcvContext) {
	count, outdir := ctx.resetWorker()
	lastWorked := 0
	for {
		obj, ok := next()
		if ok {
			if writerWorker(id, count, outdir, obj, ctx) {
				count += 1
			} else {
				ok = false
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
			errored("unable to marshal raw object", err)
			j = emptyObject
		}
		buffer.WriteString(fmt.Sprintf("\"%s\": ", e.name))
		buffer.Write(j)
	}
	return fmt.Sprintf("{%s}", buffer.String())
}

func RunReceiver() {
	config := startup()
	now := time.Now()
	ctx := &rcvContext{}
	ctx.timeFormat = now.Format("2006-01-02T15-04-05")
	ctx.output = config.Global.Output
	ctx.dump = config.Global.Dump
	go fileReceive(config)
	worker := config.Global.Workers
	i := 0
	for i < worker {
		go createWorker(i, ctx)
		i += 1
	}
	for {
		time.Sleep(1)
	}
}

func pathExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func runCollector(conf *fileConfig) {
	files := collect()
	lock.Lock()
	defer lock.Unlock()
	for _, f := range files {
		p := filepath.Join(conf.directory, f)
		if pathExists(p) {
			e := os.Remove(p)
			if e != nil {
				info(fmt.Sprintf("file error on gc: %s", p))
				errored("unable to remove garbage", e)
			}
		}
		// we are good to no longer know about this
		if _, ok := cache[f]; ok {
			delete(cache, f)
		}
	}
}

type fileConfig struct {
	directory string
	after     time.Duration
	gc        int
	sleep     time.Duration
}

func scan(conf *fileConfig) {
	files, e := ioutil.ReadDir(conf.directory)
	if e != nil {
		errored("unable to scan files", e)
		return
	}
	lock.Lock()
	defer lock.Unlock()
	requiredTime := time.Now().Add(conf.after * time.Second)
	for _, f := range files {
		n := f.Name()
		// if we already read this file we certainly should not read it again
		if _, ok := cache[n]; ok {
			continue
		}
		if f.ModTime().After(requiredTime) {
			continue
		}
		cache[n] = struct{}{}
		p := filepath.Join(conf.directory, n)
		d, e := ioutil.ReadFile(p)
		if e != nil {
			info(fmt.Sprintf("file read error: %s", p))
			errored("unable to read file", e)
			continue
		}
		queue(n, d, true)
	}
}

func fileReceive(config *Configuration) {
	conf := &fileConfig{}
	conf.directory = config.Files.Directory
	conf.gc = config.Files.Gc
	conf.sleep = time.Duration(config.Files.Sleep)
	conf.after = time.Duration(config.Files.After)
	info("file mode enabled")
	err := os.Mkdir(conf.directory, 0777)
	if err != nil {
		errored("unable to create directory (not aborting)", err)
	}
	lastCollected := 0
	for {
		if lastCollected > conf.gc {
			runCollector(conf)
			lastCollected = 0
		}
		scan(conf)
		time.Sleep(conf.sleep * time.Millisecond)
		lastCollected += 1
	}
}
