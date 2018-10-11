package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
)

func testHandlers() *handlerSettings {
	return &handlerSettings{allowEvent: true, allowDump: true, allowEmpty: true, enabled: true}
}

type writerAdjust func(*dataWriter)

type testHarness struct {
	ctx *context
	out string
	req map[string][]string
	hdl *handlerSettings
	ok  bool
	adj writerAdjust
}

func runTest(c *context, output string, r map[string][]string, h *handlerSettings, success bool) {
	test(&testHarness{ctx: c, out: output, req: r, hdl: h, ok: success})
}

func tagTest(c *context) {
	h := &testHarness{ctx: c, out: "tags", req: nil, hdl: nil, ok: true}
	h.adj = func(d *dataWriter) {
		d.objectWriter(&tagAdder{})
	}
	test(h)
}

func test(h *testHarness) {
	str := ""
	b := bytes.NewBufferString(str)
	request := h.req
	if request == nil {
		request = make(map[string][]string)
	}
	handlers := h.hdl
	if handlers == nil {
		handlers = testHandlers()
	}
	called := false
	check := func() {
		called = true
	}
	d := newDataWriter(b, check)
	if h.adj != nil {
		h.adj(d)
	}
	handle(h.ctx, request, handlers, d)
	if called != h.ok {
		panic("failed test: " + h.out)
	}
	var indent bytes.Buffer
	err := json.Indent(&indent, b.Bytes(), "", "  ")
	if err != nil {
		panic("unable to adjust output")
	}
	err = ioutil.WriteFile(h.ctx.directory+h.out, indent.Bytes(), 0644)
	if err != nil {
		panic("unable to complete test")
	}
}

func main() {
	c := &context{}
	c.directory = "bin/"
	c.limit = 10
	c.setMeta("master", "localhost")
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
	c.convert = conversions()
	delete(m, "skip")
	m["start"] = []string{"1538671495199"}
	m["end"] = []string{"1538671495201"}
	runTest(c, "startend", m, nil, true)
	// filters
	c.convert = conversions()
	delete(m, "start")
	delete(m, "end")
	c.convert["fields.simtime.raw"] = float64Conv
	filter := []string{"fields.simtime.raw:gt:100"}
	m["filter"] = filter
	runTest(c, "filters", m, nil, true)
	filter = append(filter, "id:eq:2018-10-04T12-43-25.1538671495161.2.0")
	filter = append(filter, "fields.tag.raw:eq:jzml")
	m["filter"] = filter
	runTest(c, "filtersand", m, nil, true)
	c.convert = conversions()
	tagTest(c)
}
