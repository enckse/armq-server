package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
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
	limitIndicator           = ", {\"more\": \"true\"}"
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
	// api output data
	metaFooter string
	metaHeader string
	byteHeader []byte
	byteFooter []byte
}

func conversions() map[string]typeConv {
	m := make(map[string]typeConv)
	m[tsKey] = int64Conv
	m[idKey] = strConv
	m[fmt.Sprintf("%s.%s.%s", fieldKey, tagKey, notJSON)] = strConv
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

type objectAdder interface {
	add(bool, map[string]json.RawMessage)
	done(*context, io.Writer, bool)
}

type tagAdder struct {
	objectAdder
	tracked map[string]int64
}

func (t *tagAdder) add(first bool, j map[string]json.RawMessage) {
	if first {
		t.tracked = make(map[string]int64)
	}
	o, ok := getSubField(fieldKey, j)
	if !ok {
		return
	}
	o, ok = getSubField(tagKey, o)
	if !ok {
		return
	}
	v, ok := o[notJSON]
	if !ok {
		return
	}
	tsRaw, ok := j[tsKey]
	if !ok {
		return
	}
	i, ok := int64FromJSON(tsRaw)
	if !ok {
		return
	}
	s := string(v)
	cur, ok := t.tracked[s]
	if ok {
		if i <= cur {
			return
		}
	}
	t.tracked[s] = i
}

func (t *tagAdder) done(ctx *context, w io.Writer, limit bool) {
	w.Write(ctx.byteHeader)
	first := true
	for k, v := range t.tracked {
		if !first {
			w.Write([]byte(","))
		}
		w.Write([]byte(fmt.Sprintf("{%s: %d}", k, v)))
		first = false
	}
	if limit {
		w.Write([]byte(limitIndicator))
	}
	w.Write(ctx.byteFooter)
}

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
	writer.addString(ctx.metaHeader)
	hasMore := false
	for _, p := range files {
		if count > limited {
			if has {
				hasMore = true
			}
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
		writer.addObject(!has, obj)
		has = true
		count += 1
	}
	if hasMore {
		writer.addString(limitIndicator)
	}
	writer.addString(ctx.metaFooter)
	writer.closeObjects(ctx, hasMore)
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

func (d *dataWriter) addObject(first bool, o map[string]json.RawMessage) {
	if d.object {
		d.objects.add(first, o)
	}
}

func (d *dataWriter) closeObjects(ctx *context, limited bool) {
	if d.object {
		d.objects.done(ctx, d.writer, limited)
	}
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

func getSubField(key string, j map[string]json.RawMessage) (map[string]json.RawMessage, bool) {
	v, ok := j[key]
	if !ok {
		return nil, false
	}
	var sub map[string]json.RawMessage
	err := json.Unmarshal(v, &sub)
	if err != nil {
		return nil, false
	}
	return sub, true
}

func apiMeta(ctx *context, started string) []byte {
	return []byte(fmt.Sprintf("%s {\"started\": \"%s\"} %s", ctx.metaHeader, started, ctx.metaFooter))
}

func (ctx *context) setMeta(version, host string) {
	ctx.metaHeader = "{\"meta\": {\"spec\": \"0.1\", \"api\": \"" + version + "\", \"server\": \"" + host + "\"}, \"data\": ["
	ctx.metaFooter = "]}"
	ctx.byteHeader = []byte(ctx.metaHeader)
	ctx.byteFooter = []byte(ctx.metaFooter)
}

func getConfigFalse(c *goutils.Config, key string) bool {
	if c.GetFalse(key) {
		return false
	}
	return true
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
	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}
	ctx.setMeta(vers, host)
	h := &handlerSettings{}
	h.enabled = c.GetTrue("handlers")
	h.allowEvent = getConfigFalse(c, "eventHandler")
	h.allowDump = getConfigFalse(c, "dumpHandler")
	h.allowEmpty = getConfigFalse(c, "emptyHandler")
	h.allowStart = getConfigFalse(c, "startHandler")
	h.allowReplay = getConfigFalse(c, "replayHandler")
	h.allowPlayer = getConfigFalse(c, "playerHandler")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		d := newWebDataWriter(w)
		webRequest(ctx, h, w, r, d)
	})
	apiBytes := apiMeta(ctx, time.Now().Format("2006-01-02T15:04:05"))
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w)
		w.Write(apiBytes)
	})
	http.HandleFunc("/tags", func(w http.ResponseWriter, r *http.Request) {
		obj := newWebDataWriter(w)
		obj.objectWriter(&tagAdder{})
		webRequest(ctx, h, w, r, obj)
	})
	err = http.ListenAndServe(bind, nil)
	if err != nil {
		goutils.Fatal("unable to do http serve", err)
	}
}
