package receiver

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

	"voidedtech.com/armq-server/internal"
)

var (
	emptyObject = []byte("{}")
	gcLock      = &sync.Mutex{}
	readLock    = &sync.Mutex{}
	objcache    = []*object{}
	gc          = []string{}
	lock        = &sync.Mutex{}
	cache       = make(map[string]struct{})
)

const (
	delimiter     = "`"
	sleepCycleMin = 90
	sleepCycleMax = 108
)

type (
	fileConfig struct {
		directory string
		after     time.Duration
		gc        int
		sleep     time.Duration
	}

	// Datum is representative output from armq
	Datum struct {
		ID        string
		Timestamp int64
		Version   string
		File      string
		Date      string
	}

	object struct {
		id   string
		data []byte
		gc   bool
	}
)

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

func (d *Datum) toJSON() string {
	return fmt.Sprintf("\"%s\": \"%s\", \"%s\": %d, \"vers\": \"%s\", \"file\": \"%s\", \"%s\": \"%s\"", internal.IDKey, d.ID, internal.TSKey, d.Timestamp, d.Version, d.File, internal.DTKey, d.Date)
}

func writerWorker(id, count int, outdir string, obj *object, conf *internal.Configuration, timeStr string) bool {
	dump := &internal.Entry{Raw: string(obj.data), Type: internal.NotJSON}
	datum := &Datum{}
	parts := strings.Split(dump.Raw, delimiter)
	ts := parts[0]
	i, e := strconv.ParseInt(ts, 10, 64)
	if e != nil {
		internal.Info(fmt.Sprintf("unable to parse timestamp (not critical): %s", obj.id))
		internal.Errored("parse error was", e)
		i = -1
	}
	datum.Timestamp = i
	datum.Date = time.Unix(i/1000, 0).Format("2006-01-02T15:04:05")
	datum.Version = parts[1]
	datum.File = obj.id
	datum.ID = fmt.Sprintf("%s.%d.%d.%d", timeStr, datum.Timestamp, id, count)
	fields := detectJSON(parts[2:])
	if fields == "" {
		fields = "{}"
	}
	j := emptyObject
	if conf.Global.Dump {
		j, e = json.Marshal(dump)
		if e != nil {
			internal.Info(fmt.Sprintf("unable to handle file %s", obj.id))
			internal.Errored("unable to read object to json", e)
			return false
		}
	}
	j = []byte(fmt.Sprintf("{%s, \"%s\": %s, \"%s\": %s}", datum.toJSON(), internal.DumpKey, j, internal.FieldKey, fields))
	p := filepath.Join(outdir, datum.ID)
	if err := ioutil.WriteFile(p, j, 0644); err != nil {
		internal.Info(fmt.Sprintf("error saving results: %s", p))
		internal.Errored("unable to save file", err)
		return false
	}
	return true
}

func resetWorker(conf *internal.Configuration) (int, string) {
	now := time.Now().Format("2006-01-02")
	p := filepath.Join(conf.Global.Output, now)
	if !internal.PathExists(p) {
		if err := os.MkdirAll(p, 0755); err != nil {
			internal.Info(fmt.Sprintf("error reseting path: %s", p))
			internal.Errored("error for path reset", err)
		}
	}
	return 0, p
}

func createWorker(id int, conf *internal.Configuration, timeStr string) {
	count, outdir := resetWorker(conf)
	lastWorked := 0
	for {
		obj, ok := next()
		if ok {
			if writerWorker(id, count, outdir, obj, conf, timeStr) {
				count++
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
				count, outdir = resetWorker(conf)
			case id > 0 && lastWorked >= sleepCycleMax:
				// initial worker can never go this slow
				cooldown = 30
			}
			if lastWorked < sleepCycleMax {
				lastWorked++
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
	entries := []*internal.Entry{}
	for idx, section := range segment {
		p := &internal.Entry{}
		p.Type = internal.NotJSON
		p.Raw = section
		var arr []json.RawMessage
		bytes := []byte(section)
		if json.Unmarshal(bytes, &arr) == nil {
			p.Array = arr
			p.Type = internal.ArrayJSON
		} else {
			var obj map[string]json.RawMessage
			if json.Unmarshal(bytes, &obj) == nil {
				p.Type = internal.ObjJSON
				p.Object = obj
			}
		}
		p.Name = fmt.Sprintf("%s%d", internal.FKey, idx)
		entries = append(entries, p)
	}
	var buffer bytes.Buffer
	for idx, e := range entries {
		if idx > 0 {
			buffer.WriteString(",")
		}
		entry := &internal.Entry{Type: e.Type}
		switch e.Type {
		case internal.NotJSON:
			entry.Raw = e.Raw
		case internal.ArrayJSON:
			entry.Array = e.Array
		case internal.ObjJSON:
			entry.Object = e.Object
		}
		j, err := json.Marshal(entry)
		if err != nil {
			internal.Errored("unable to marshal raw object", err)
			j = emptyObject
		}
		buffer.WriteString(fmt.Sprintf("\"%s\": ", e.Name))
		buffer.Write(j)
	}
	return fmt.Sprintf("{%s}", buffer.String())
}

// Run runs the receiving component to parse armq outputs
func Run(vers string) {
	config, _ := internal.Startup(vers)
	go fileReceive(config)
	worker := config.Global.Workers
	i := 0
	n := internal.Now()
	for i < worker {
		go createWorker(i, config, n)
		i++
	}
	for {
		time.Sleep(1)
	}
}

func runCollector(conf *fileConfig) {
	files := collect()
	lock.Lock()
	defer lock.Unlock()
	for _, f := range files {
		p := filepath.Join(conf.directory, f)
		if internal.PathExists(p) {
			e := os.Remove(p)
			if e != nil {
				internal.Info(fmt.Sprintf("file error on gc: %s", p))
				internal.Errored("unable to remove garbage", e)
			}
		}
		// we are good to no longer know about this
		if _, ok := cache[f]; ok {
			delete(cache, f)
		}
	}
}

func scan(conf *fileConfig) {
	files, e := ioutil.ReadDir(conf.directory)
	if e != nil {
		internal.Errored("unable to scan files", e)
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
			internal.Info(fmt.Sprintf("file read error: %s", p))
			internal.Errored("unable to read file", e)
			continue
		}
		queue(n, d, true)
	}
}

func fileReceive(config *internal.Configuration) {
	conf := &fileConfig{}
	conf.directory = config.Files.Directory
	conf.gc = config.Files.Gc
	conf.sleep = time.Duration(config.Files.Sleep)
	conf.after = time.Duration(config.Files.After)
	internal.Info("file mode enabled")
	if err := os.Mkdir(conf.directory, 0777); err != nil {
		internal.Errored("unable to create directory (not aborting)", err)
	}
	lastCollected := 0
	for {
		if lastCollected > conf.gc {
			runCollector(conf)
			lastCollected = 0
		}
		scan(conf)
		time.Sleep(conf.sleep * time.Millisecond)
		lastCollected++
	}
}
