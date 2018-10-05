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
}
