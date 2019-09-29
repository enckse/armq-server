package internal

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"time"

	"voidedtech.com/armq-server/internal/common"
)

func testHandlers() *common.Configuration {
	cfg := &common.Configuration{}
	cfg.API.Handlers.Enable = true
	cfg.API.Handlers.Dump = true
	cfg.API.Handlers.Empty = true
	cfg.API.Handlers.Event = true
	return cfg
}

type (
	writerAdjust func(*DataWriter)

	testHarness struct {
		ctx *Context
		out string
		req map[string][]string
		hdl *common.Configuration
		ok  bool
		adj writerAdjust
	}
)

func runTest(c *Context, output string, r map[string][]string, h *common.Configuration, success bool) {
	test(&testHarness{ctx: c, out: output, req: r, hdl: h, ok: success})
}

func tagTest(c *Context) {
	h := &testHarness{ctx: c, out: "tags", req: nil, hdl: nil, ok: true}
	h.adj = func(d *DataWriter) {
		d.objectWriter(&TagAdder{})
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
	d := NewDataWriter(b, check)
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
	err = ioutil.WriteFile(h.ctx.Directory+h.out, indent.Bytes(), 0644)
	if err != nil {
		panic("unable to complete test")
	}
}

// RunTests performs test execution
func RunTests() {
	c := &Context{}
	c.Directory = "bin/"
	c.Limit = 10
	c.ScanStart = -10 * 24 * time.Hour
	c.ScanEnd = 24 * time.Hour
	c.SetMeta("master", "localhost")
	runTest(c, "normal", nil, nil, true)
	runTest(c, "nohandlers", nil, &common.Configuration{}, true)
	// limit input
	m := make(map[string][]string)
	m["limit"] = []string{"1"}
	runTest(c, "limit", m, nil, true)
	// skip input
	delete(m, "limit")
	c.Limit = 1
	m["skip"] = []string{"1"}
	runTest(c, "skip", m, nil, true)
	// start & end
	c.Limit = 10
	c.Convert = DefaultConverters()
	delete(m, "skip")
	m["start"] = []string{"1538671495199"}
	m["end"] = []string{"1538671495201"}
	runTest(c, "startend", m, nil, true)
	// filters
	c.Convert = DefaultConverters()
	delete(m, "start")
	delete(m, "end")
	c.Convert["fields.simtime.raw"] = float64Conv
	filter := []string{"fields.simtime.raw:gt:100"}
	m["filter"] = filter
	runTest(c, "filters", m, nil, true)
	filter = append(filter, "id:eq:2018-10-04T12-43-25.1538671495161.2.0")
	filter = append(filter, "fields.tag.raw:eq:jzml")
	m["filter"] = filter
	runTest(c, "filtersand", m, nil, true)
	c.Convert = DefaultConverters()
	tagTest(c)
}
