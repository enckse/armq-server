package internal

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

	"voidedtech.com/armq-server/internal/common"
)

const (
	maxOp                           = 5
	minOp                           = -1
	int64Conv       common.TypeConv = 1
	strConv         common.TypeConv = 2
	intConv         common.TypeConv = 3
	float64Conv     common.TypeConv = 4
	filterDelimiter                 = ":"
	startStringOp                   = "ge"
	endStringOp                     = "le"
	limitIndicator                  = ", {\"limited\": \"true\"}"
	spec                            = "0.1"
	notJSON                         = "raw"
	objJSON                         = "object"
	arrayJSON                       = "array"
	emptyJSON                       = "empty"
	fieldKey                        = "fields"
	dumpKey                         = "dump"
	fKey                            = "field"
	tsKey                           = "ts"
	idKey                           = "id"
	tagKey                          = "tag"
	dtKey                           = "dt"
	field0Key                       = fKey + "0"
	field1Key                       = fKey + "1"
	field2Key                       = fKey + "2"
	field3Key                       = fKey + "3"
	field4Key                       = fKey + "4"
	field5Key                       = fKey + "5"
	lessThan        common.OpType   = 0
	equals          common.OpType   = 1
	lessTE          common.OpType   = 2
	greatThan       common.OpType   = 3
	greatTE         common.OpType   = 4
	nEquals         common.OpType   = maxOp
	invalidOp       common.OpType   = minOp
	// TSKey is the timestamp key
	TSKey = tsKey
	// IDKey is the identifier key
	IDKey = idKey
	// DumpKey is the raw dump
	DumpKey = dumpKey
	// DTKey is the datetime key
	DTKey = dtKey
	// NotJSON indicates raw json-ish object
	NotJSON = notJSON
	// FieldKey is for the fields in the data
	FieldKey = fieldKey
	// ArrayJSON indicates it is an array of things
	ArrayJSON = arrayJSON
	// ObjJSON indicates it is a json-ic ojbect
	ObjJSON = objJSON
	// FKey is a field indicator
	FKey = fKey
)

type (
	dataFilter struct {
		field      string
		op         common.OpType
		int64Val   int64
		strVal     string
		intVal     int
		float64Val float64
		fxn        common.TypeConv
	}

	apiContext struct {
		limit     int
		directory string
		convert   map[string]common.TypeConv
		// api output data
		metaFooter string
		metaHeader string
		byteHeader []byte
		byteFooter []byte
		// how we scan for data
		scanStart time.Duration
		scanEnd   time.Duration
	}

	onHeaders func()

	objectAdder interface {
		add(bool, map[string]json.RawMessage)
		done(*apiContext, io.Writer, bool)
	}

	tagMeta struct {
		endTime      int64
		endTimeStr   string
		startTime    int64
		startTimeStr string
	}

	tagAdder struct {
		objectAdder
		tracked map[string]*tagMeta
	}

	dataWriter struct {
		writer  io.Writer
		write   bool
		headers onHeaders
		header  bool
		objects objectAdder
		object  bool
		limit   bool
	}
)

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

func conversions() map[string]common.TypeConv {
	m := make(map[string]common.TypeConv)
	m[tsKey] = int64Conv
	m[idKey] = strConv
	m[fmt.Sprintf("%s.%s.%s", fieldKey, tagKey, notJSON)] = strConv
	return m
}

func stringToOp(op string) common.OpType {
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

func parseFilter(filter string, mapping map[string]common.TypeConv) *dataFilter {
	parts := strings.Split(filter, filterDelimiter)
	if len(parts) < 3 {
		common.Info("filter missing components")
		return nil
	}
	val := strings.Join(parts[2:], filterDelimiter)
	f := &dataFilter{}
	f.field = parts[0]
	t, ok := mapping[f.field]
	if !ok {
		common.Info(fmt.Sprintf("filter field unknown: %s", f.field))
		return nil
	}
	f.op = stringToOp(parts[1])
	if f.op == invalidOp {
		common.Info("filter op invalid")
		return nil
	}
	f.fxn = t
	switch t {
	case intConv:
		i, e := strconv.Atoi(val)
		if e != nil {
			common.Info("filter is not an int")
			return nil
		}
		f.intVal = i
	case int64Conv:
		i, e := strconv.ParseInt(val, 10, 64)
		if e != nil {
			common.Info("filter is not an int64")
			return nil
		}
		f.int64Val = i
	case float64Conv:
		i, e := strconv.ParseFloat(val, 64)
		if e != nil {
			common.Info("filter is not a float64")
		}
		f.float64Val = i
	case strConv:
		if f.op == equals || f.op == nEquals {
			f.strVal = val
		} else {
			common.Info("filter string op is invalid")
			return nil
		}
	default:
		common.Info("unknown filter type")
		return nil
	}
	return f
}

func timeFilter(op, value string, mapping map[string]common.TypeConv) *dataFilter {
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

func (t *tagAdder) add(first bool, j map[string]json.RawMessage) {
	if first {
		t.tracked = make(map[string]*tagMeta)
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
	dtRaw, ok := j[dtKey]
	if !ok {
		return
	}
	d, ok := stringFromJSON(dtRaw)
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
		if i >= cur.endTime {
			cur.endTime = i
			cur.endTimeStr = d
		} else {
			if i <= cur.startTime {
				cur.startTime = i
				cur.startTimeStr = d
			} else {
				return
			}
		}
		t.tracked[s] = cur
	} else {
		t.tracked[s] = &tagMeta{endTime: i, endTimeStr: d, startTime: i, startTimeStr: d}
	}
}

func (t *tagAdder) done(ctx *apiContext, w io.Writer, limit bool) {
	w.Write(ctx.byteHeader)
	first := true
	for k, v := range t.tracked {
		if !first {
			w.Write([]byte(","))
		}
		w.Write([]byte(fmt.Sprintf("{%s: [%d, \"%s\", %d, \"%s\"]}", k, v.startTime, v.startTimeStr, v.endTime, v.endTimeStr)))
		first = false
	}
	if limit {
		w.Write([]byte(limitIndicator))
	}
	w.Write(ctx.byteFooter)
}

func newDataWriter(w io.Writer, h onHeaders) *dataWriter {
	o := &dataWriter{}
	o.write = w != nil
	o.writer = w
	o.header = h != nil
	o.headers = h
	o.limit = true
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

func handle(ctx *apiContext, req map[string][]string, h *common.Configuration, writer *dataWriter) bool {
	dataFilters := []*dataFilter{}
	limited := 0
	if writer.limit {
		limited = ctx.limit
	}
	skip := 0
	startDate := ""
	endDate := ""
	fileRead := ""
	seek := false
	for k, p := range req {
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
			if err == nil {
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
		case "seek":
			seek = true
		}
	}
	stime := getDate(startDate, ctx.scanStart)
	etime := getDate(endDate, ctx.scanEnd)
	dirs, e := ioutil.ReadDir(ctx.directory)
	if seek && len(dirs) > 0 {
		last := dirs[len(dirs)-1]
		dirs = []os.FileInfo{last}
	}
	if e != nil {
		common.Errored("unable to read dir", e)
		return false
	}
	filterFiles := len(fileRead) > 0
	files := []string{}
	for _, d := range dirs {
		dname := d.Name()
		if d.IsDir() {
			mtime := d.ModTime()
			if !seek {
				if mtime.Before(stime) || mtime.After(etime) {
					continue
				}
			}
			p := filepath.Join(ctx.directory, dname)
			f, e := ioutil.ReadDir(p)
			if e != nil {
				common.Info(fmt.Sprintf("unable to read subdir: %s", dname))
				common.Errored("reading subdir failed", e)
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
		if limited > 0 && count > limited {
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
							common.Info(fmt.Sprintf("unable to unmarshal obj: %s (%s)", p, d.field))
							common.Errored("unmarshal error", err)
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
		if has {
			writer.addString(",")
		}
		writer.add(b)
		writer.addObject(!has, obj)
		has = true
		count++
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

func newWebdataWriter(w http.ResponseWriter) *dataWriter {
	return newDataWriter(w, func() {
		writeSuccess(w)
	})
}

func (d *dataWriter) addObject(first bool, o map[string]json.RawMessage) {
	if d.object {
		d.objects.add(first, o)
	}
}

func (d *dataWriter) closeObjects(ctx *apiContext, limited bool) {
	if d.object {
		d.objects.done(ctx, d.writer, limited)
	}
}

func webRequest(ctx *apiContext, h *common.Configuration, w http.ResponseWriter, r *http.Request, d *dataWriter) {
	success := handle(ctx, r.URL.Query(), h, d)
	if !success {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (d *dataWriter) objectWriter(adder objectAdder) {
	d.object = true
	d.write = false
	d.objects = adder
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

func apiMeta(ctx *apiContext, started string) []byte {
	return []byte(fmt.Sprintf("%s {\"started\": \"%s\"} %s", ctx.metaHeader, started, ctx.metaFooter))
}

func (ctx *apiContext) setMeta(version, host string) {
	ctx.metaHeader = "{\"meta\": {\"spec\": \"" + spec + "\", \"api\": \"" + version + "\", \"server\": \"" + host + "\"}, \"data\": ["
	ctx.metaFooter = "]}"
	ctx.byteHeader = []byte(ctx.metaHeader)
	ctx.byteFooter = []byte(ctx.metaFooter)
}

// Run runs the API listener
func Run(vers string) {
	conf := common.Startup(vers)
	dir := conf.Global.Output
	bind := conf.API.Bind
	limit := conf.API.Limit
	ctx := &apiContext{}
	ctx.limit = limit
	ctx.directory = dir
	ctx.convert = conversions()
	ctx.scanStart = time.Duration(conf.API.StartScan) * 24 * time.Hour
	ctx.scanEnd = time.Duration(conf.API.EndScan) * 24 * time.Hour
	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}
	ctx.setMeta(vers, host)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		d := newWebdataWriter(w)
		webRequest(ctx, conf, w, r, d)
	})
	apiBytes := apiMeta(ctx, time.Now().Format("2006-01-02T15:04:05"))
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w)
		w.Write(apiBytes)
	})
	http.HandleFunc("/tags", func(w http.ResponseWriter, r *http.Request) {
		obj := newWebdataWriter(w)
		obj.limit = false
		obj.objectWriter(&tagAdder{})
		webRequest(ctx, conf, w, r, obj)
	})
	err = http.ListenAndServe(bind, nil)
	if err != nil {
		common.Errored("unable to do http serve", err)
		panic("unable to host")
	}
}
