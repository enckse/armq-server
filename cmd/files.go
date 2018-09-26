package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/epiphyte/goutils"
)

const (
	// 50 = 5 seconds
	gcNow = 50
)

var (
	lock  = &sync.Mutex{}
	cache = make(map[string]struct{})
)

func runCollector(dir string) {
	files := collect()
	lock.Lock()
	defer lock.Unlock()
	for _, f := range files {
		p := filepath.Join(dir, f)
		goutils.WriteDebug("collecting", p)
		_, e := goutils.RunBashCommand(fmt.Sprintf("rm -f %s", p))
		if e != nil {
			goutils.WriteWarn("file error on gc", p)
			goutils.WriteError("unable to remove garbage", e)
		}
		// we are good to no longer know about this
		if _, ok := cache[f]; ok {
			delete(cache, f)
		}
	}
}

func scan(dir string) {
	files, e := ioutil.ReadDir(dir)
	if e != nil {
		goutils.WriteError("unable to scan files", e)
		return
	}
	lock.Lock()
	defer lock.Unlock()
	requiredTime := time.Now().Add(-10 * time.Second)
	for _, f := range files {
		n := f.Name()
		// if we already read this file we certainly should not read it again
		if _, ok := cache[n]; ok {
			continue
		}
		if f.ModTime().After(requiredTime) {
			continue
		}
		goutils.WriteDebug("reading", n)
		cache[n] = struct{}{}
		p := filepath.Join(dir, n)
		d, e := ioutil.ReadFile(p)
		if e != nil {
			goutils.WriteWarn("file read error", p)
			goutils.WriteError("unable to read file", e)
			continue
		}
		queue(n, d, true)
	}
}

func fileReceive(ctx *context) {
	goutils.WriteInfo("file mode enabled")
	err := os.Mkdir(ctx.directory, 0777)
	if err != nil {
		goutils.WriteError("unable to create directory (not aborting)", err)
	}
	lastCollected := 0
	for {
		if lastCollected > gcNow {
			goutils.WriteDebug("collecting garbage")
			runCollector(ctx.directory)
			lastCollected = 0
		}
		scan(ctx.directory)
		time.Sleep(100 * time.Millisecond)
		lastCollected += 1
	}
}
