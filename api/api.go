package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/epiphyte/goutils"
)

const (
	int64Conv       typeConv = 1
	strConv         typeConv = 2
	intConv         typeConv = 3
	float64Conv     typeConv = 4
	filterDelimiter          = ":"
	startStringOp            = "ge"
	endStringOp              = "le"
)

type dataFilter struct {
	field      string
	op         opType
	int64Val   int64
	strVal     string
	intVal     int
	float64Val float64
	fxn        typeConv
}

func (f *dataFilter) check(d []byte) bool {
	switch f.fxn {
	case int64Conv:
		return int64Converter(f.int64Val, d, f.op)
	case intConv:
		return intConverter(f.intVal, d, f.op)
	case strConv:
		return stringConverter(f.strVal, d, f.op)
	case float64Conv:
		return float64Converter(f.float64Val, d, f.op)
	}
	return false
}

type context struct {
	limit     int
	directory string
	convert   map[string]typeConv
}

func conversions() map[string]typeConv {
	m := make(map[string]typeConv)
	m[tsKey] = int64Conv
	m[idKey] = strConv
	return m
}

func stringToOp(op string) opType {
	switch op {
	case "eq":
		return equals
	case "neq":
		return nEquals
	case "gt":
		return greatThan
	case "lt":
		return lessThan
	case endStringOp:
		return lessTE
	case startStringOp:
		return greatTE
	}
	return invalidOp
}

func parseFilter(filter string, mapping map[string]typeConv) *dataFilter {
	parts := strings.Split(filter, filterDelimiter)
	if len(parts) < 3 {
		goutils.WriteWarn("filter missing components")
		return nil
	}
	val := strings.Join(parts[2:], filterDelimiter)
	f := &dataFilter{}
	f.field = parts[0]
	t, ok := mapping[f.field]
	if !ok {
		goutils.WriteWarn("filter field unknown", f.field)
		return nil
	}
	f.op = stringToOp(parts[1])
	if f.op == invalidOp {
		goutils.WriteWarn("filter op invalid")
		return nil
	}
	f.fxn = t
	switch t {
	case intConv:
		i, e := strconv.Atoi(val)
		if e != nil {
			goutils.WriteWarn("filter is not an int")
			return nil
		}
		f.intVal = i
	case int64Conv:
		i, e := strconv.ParseInt(val, 10, 64)
		if e != nil {
			goutils.WriteWarn("filter is not an int64")
			return nil
		}
		f.int64Val = i
	case float64Conv:
		i, e := strconv.ParseFloat(val, 64)
		if e != nil {
			goutils.WriteWarn("filter is not a float64")
		}
		f.float64Val = i
	case strConv:
		if f.op == equals || f.op == nEquals {
			f.strVal = val
		} else {
			goutils.WriteWarn("filter string op is invalid")
			return nil
		}
	default:
		goutils.WriteWarn("unknown filter type")
		return nil
	}
	return f
}

func timeFilter(op, value string, mapping map[string]typeConv) *dataFilter {
	return parseFilter(fmt.Sprintf("%s%s%s%s%s", tsKey, filterDelimiter, op, filterDelimiter, value), mapping)
}

func getDate(in string, adding time.Duration) time.Time {
	var t time.Time
	if in == "" {
		t = time.Now().Add(adding)
	} else {
		t, _ = time.Parse("2006-01-02", in)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

type onHeaders func()
type objectAdder func(io.Writer, map[string]json.RawMessage)

type dataWriter struct {
	writer  io.Writer
	write   bool
	headers onHeaders
	header  bool
	objects objectAdder
	object  bool
}

func newDataWriter(w io.Writer, h onHeaders) *dataWriter {
	o := &dataWriter{}
	o.write = w != nil
	o.writer = w
	o.header = h != nil
	o.headers = h
	return o
}

func (d *dataWriter) setHeaders() {
	if d.header {
		d.headers()
	}
}

func (d *dataWriter) add(b []byte) {
	if d.write {
		d.writer.Write(b)
	}
}

func (d *dataWriter) addString(s string) {
	d.add([]byte(s))
}

func handle(ctx *context, req map[string][]string, h *handlerSettings, writer *dataWriter) bool {
	dataFilters := []*dataFilter{}
	limited := ctx.limit
	skip := 0
	startDate := ""
	endDate := ""
	fileRead := ""
	for k, p := range req {
		goutils.WriteDebug(k, p...)
		if len(p) == 0 {
			continue
		}
		switch k {
		case "filter":
			for _, val := range p {
				f := parseFilter(val, ctx.convert)
				if f != nil {
					dataFilters = append(dataFilters, f)
				}
			}

		case "start":
			fallthrough
		case "end":
			mode := endStringOp
			if k == "start" {
				mode = startStringOp
			}
			f := timeFilter(mode, p[0], ctx.convert)
			if f != nil {
				dataFilters = append(dataFilters, f)
			}
		case "limit":
			i, err := strconv.Atoi(p[0])
			if err == nil && i > 0 {
				limited = i
			}
		case "skip":
			i, err := strconv.Atoi(p[0])
			if err == nil && i > 0 {
				skip = i
			}
		case "files":
			fileRead = strings.TrimSpace(p[0])
		case "startdate":

			startDate = strings.TrimSpace(p[0])
		case "enddate":
			endDate = strings.TrimSpace(p[0])
		}
	}
	stime := getDate(startDate, -10*24*time.Hour)
	etime := getDate(endDate, 24*time.Hour)
	dirs, e := ioutil.ReadDir(ctx.directory)
	if e != nil {
		goutils.WriteError("unable to read dir", e)
		return false
	}
	filterFiles := len(fileRead) > 0
	goutils.WriteDebug("file filter", fileRead)
	files := []string{}
	for _, d := range dirs {
		if d.IsDir() {
			mtime := d.ModTime()
			if mtime.Before(stime) || mtime.After(etime) {
				continue
			}
			dname := d.Name()
			p := filepath.Join(ctx.directory, dname)
			f, e := ioutil.ReadDir(p)
			if e != nil {
				goutils.WriteWarn("unable to read subdir", dname)
				goutils.WriteError("reading subdir failed", e)
				continue
			}
			for _, file := range f {
				name := file.Name()
				if filterFiles {
					if !strings.HasPrefix(name, fileRead) {
						continue
					}
				}
				files = append(files, filepath.Join(p, name))
			}
		}
	}

	count := 0
	has := false
	writer.setHeaders()
	writer.addString("{\"data\": [")
	for _, p := range files {
		if count > limited {
			break
		}
		obj, b := loadFile(p, h)
		if len(dataFilters) > 0 {
			valid := false
			for _, d := range dataFilters {
				filterObj := obj
				parts := strings.Split(d.field, ".")
				fieldLen := len(parts) - 1
				for i, p := range parts {
					v, ok := filterObj[p]
					if !ok {
						break

					}
					if i == fieldLen {
						valid = d.check(v)
						break
					} else {
						var sub map[string]json.RawMessage
						err := json.Unmarshal(v, &sub)
						if err != nil {
							goutils.WriteWarn("unable to unmarshal obj", p, d.field)
							goutils.WriteError("unmarshal error", err)
							break
						}
						filterObj = sub
					}
				}
				if !valid {
					break
				}
			}
			if !valid {
				continue
			}
		}
		if skip > 0 {
			skip += -1
			continue
		}
		goutils.WriteDebug("passed", p)
		if has {
			writer.addString(",")
		}
		writer.add(b)
		has = true
		count += 1
	}
	writer.addString("]}")
	return true
}

func writeSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func newWebDataWriter(w http.ResponseWriter) *dataWriter {
	return newDataWriter(w, func() {
		writeSuccess(w)
	})
}

func webRequest(ctx *context, h *handlerSettings, w http.ResponseWriter, r *http.Request, d *dataWriter) {
	success := handle(ctx, r.URL.Query(), h, d)
	if !success {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (o *dataWriter) objectWriter(adder objectAdder) {
	o.object = true
	o.write = false
	o.objects = adder
}

func tagWriter(w io.Writer, j map[string]json.RawMessage) {
}

func runApp() {
	conf := startup()
	dir := conf.GetStringOrDefault(outKey, dataDir)
	c := conf.GetSection("[api]")
	bind := c.GetStringOrDefault("bind", ":8080")
	limit := c.GetIntOrDefaultOnly("limit", 1000)
	goutils.WriteDebug("api ready")
	ctx := &context{}
	ctx.limit = limit
	ctx.directory = dir
	ctx.convert = conversions()
	h := &handlerSettings{}
	h.enabled = c.GetTrue("handlers")
	h.allowEvent = true
	h.allowDump = true
	h.allowEmpty = true
	if c.GetFalse("eventHander") {
		h.allowEvent = false
	}
	if c.GetFalse("dumpHandler") {
		h.allowDump = false
	}
	if c.GetFalse("emptyHandler") {
		h.allowEmpty = false
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		d := newWebDataWriter(w)
		webRequest(ctx, h, w, r, d)
	})
	http.HandleFunc("/tags", func(w http.ResponseWriter, r *http.Request) {
		obj := newWebDataWriter(w)
		obj.objectWriter(tagWriter)
		webRequest(ctx, h, w, r, obj)
	})
	err := http.ListenAndServe(bind, nil)
	if err != nil {
		goutils.Fatal("unable to do http serve", err)
	}
}
