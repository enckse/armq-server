package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/epiphyte/goutils/config"
	"github.com/epiphyte/goutils/logger"
	"github.com/epiphyte/goutils/opsys"
)

var (
	lock  = &sync.Mutex{}
	cache = make(map[string]struct{})
)

func runCollector(conf *fileConfig) {
	files := collect()
	lock.Lock()
	defer lock.Unlock()
	for _, f := range files {
		p := filepath.Join(conf.directory, f)
		logger.WriteDebug("collecting", p)
		_, e := opsys.RunBashCommand(fmt.Sprintf("rm -f %s", p))
		if e != nil {
			logger.WriteWarn("file error on gc", p)
			logger.WriteError("unable to remove garbage", e)
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
		logger.WriteError("unable to scan files", e)
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
		logger.WriteDebug("reading", n)
		cache[n] = struct{}{}
		p := filepath.Join(conf.directory, n)
		d, e := ioutil.ReadFile(p)
		if e != nil {
			logger.WriteWarn("file read error", p)
			logger.WriteError("unable to read file", e)
			continue
		}
		queue(n, d, true)
	}
}

func fileReceive(config *config.Config) {
	conf := &fileConfig{}
	conf.directory = config.GetStringOrDefault("directory", "/opt/armq/")
	conf.gc = config.GetIntOrDefaultOnly("gc", 50)
	conf.sleep = time.Duration(config.GetIntOrDefaultOnly("sleep", 100))
	conf.after = time.Duration(config.GetIntOrDefaultOnly("after", -10))
	logger.WriteInfo("file mode enabled")
	err := os.Mkdir(conf.directory, 0777)
	if err != nil {
		logger.WriteError("unable to create directory (not aborting)", err)
	}
	lastCollected := 0
	for {
		if lastCollected > conf.gc {
			logger.WriteDebug("collecting garbage")
			runCollector(conf)
			lastCollected = 0
		}
		scan(conf)
		time.Sleep(conf.sleep * time.Millisecond)
		lastCollected += 1
	}
}
