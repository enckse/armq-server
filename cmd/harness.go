package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
)

func main() {
	mainTest()
}

func testHandlers() *handlerSettings {
	return &handlerSettings{allowEvent: true, allowDump: true, allowEmpty: true, enabled: true}
}

func runTest(c *context, output string, req map[string][]string, h *handlerSettings, success bool) {
	str := ""
	b := bytes.NewBufferString(str)
	request := req
	if request == nil {
		request = make(map[string][]string)
	}
	handlers := h
	if handlers == nil {
		handlers = testHandlers()
	}
	called := false
	check := func() {
		called = true
	}
	handle(c, b, request, handlers, check)
	if called != success {
		panic("failed test: " + output)
	}
	var indent bytes.Buffer
	err := json.Indent(&indent, b.Bytes(), "", "  ")
	if err != nil {
		panic("unable to adjust output")
	}
	err = ioutil.WriteFile(c.directory+output, indent.Bytes(), 0644)
	if err != nil {
		panic("unable to complete test")
	}
}

func mainTest() {
	c := &context{}
	c.directory = "bin/"
	c.limit = 10
	runTest(c, "normal", nil, nil, true)
	runTest(c, "nohandlers", nil, &handlerSettings{}, true)
	// limit input
	m := make(map[string][]string)
	m["limit"] = []string{"1"}
	runTest(c, "limit", m, nil, true)
	// skip input
	delete(m, "limit")
	c.limit = 1
	m["skip"] = []string{"1"}
	runTest(c, "skip", m, nil, true)
	// start & end
	c.limit = 10
	c.convert = make(map[string]typeConv)
	c.convert["ts"] = int64Conv
	delete(m, "skip")
	m["start"] = []string{"1538671495199"}
	m["end"] = []string{"1538671495201"}
	runTest(c, "startend", m, nil, true)
	// filters
	c.convert = make(map[string]typeConv)
	delete(m, "start")
	delete(m, "end")
	c.convert["fields.simtime.raw"] = float64Conv
	c.convert["id"] = strConv
	filter := []string{"fields.simtime.raw:gt:100"}
	m["filter"] = filter
	runTest(c, "filters", m, nil, true)
}
