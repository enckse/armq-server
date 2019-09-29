package api

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

	"voidedtech.com/armq-server/internal"
	"voidedtech.com/armq-server/internal/messages"
)

const (
	// Int64Conv for int64 conversions
	Int64Conv internal.TypeConv = 1
	// StrConv for string conversions
	StrConv internal.TypeConv = 2
	// IntConv for integer conversions
	IntConv internal.TypeConv = 3
	// Float64Conv for float64 conversions
	Float64Conv     internal.TypeConv = 4
	filterDelimiter                   = ":"
	startStringOp                     = "ge"
	endStringOp                       = "le"
	limitIndicator                    = ", {\"limited\": \"true\"}"
	spec                              = "0.1"
)

type (
	dataFilter struct {
		field      string
		op         internal.OpType
		int64Val   int64
		strVal     string
		intVal     int
		float64Val float64
		fxn        internal.TypeConv
	}

	// Context represents operating context
	Context struct {
		Limit     int
		Directory string
		Convert   map[string]internal.TypeConv
		// api output data
		metaFooter string
		metaHeader string
		byteHeader []byte
		byteFooter []byte
		// how we scan for data
		ScanStart time.Duration
		ScanEnd   time.Duration
	}

	onHeaders func()

	objectAdder interface {
		add(bool, map[string]json.RawMessage)
		done(*Context, io.Writer, bool)
	}

	tagMeta struct {
		endTime      int64
		endTimeStr   string
		startTime    int64
		startTimeStr string
	}

	// TagAdder handles tagged results
	TagAdder struct {
		objectAdder
		tracked map[string]*tagMeta
	}

	// DataWriter handles writing data responses
	DataWriter struct {
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
	case Int64Conv:
		return internal.JSONint64Converter(f.int64Val, d, f.op)
	case IntConv:
		return internal.JSONintConverter(f.intVal, d, f.op)
	case StrConv:
		return internal.JSONstringConverter(f.strVal, d, f.op)
	case Float64Conv:
		return internal.JSONfloat64Converter(f.float64Val, d, f.op)
	}
	return false
}

// DefaultConverters initializes the converts we should plan to use
func DefaultConverters() map[string]internal.TypeConv {
	return map[string]internal.TypeConv{
		internal.TSKey: Int64Conv,
		internal.IDKey: StrConv,
		fmt.Sprintf("%s.%s.%s", internal.FieldKey, internal.TagKey, internal.NotJSON): StrConv,
	}
}

func stringToOp(op string) internal.OpType {
	switch op {
	case "eq":
		return internal.Equals
	case "neq":
		return internal.NEquals
	case "gt":
		return internal.GreatThan
	case "lt":
		return internal.LessThan
	case endStringOp:
		return internal.LessTE
	case startStringOp:
		return internal.GreatTE
	}
	return internal.InvalidOp
}

func parseFilter(filter string, mapping map[string]internal.TypeConv) *dataFilter {
	parts := strings.Split(filter, filterDelimiter)
	if len(parts) < 3 {
		internal.Info("filter missing components")
		return nil
	}
	val := strings.Join(parts[2:], filterDelimiter)
	f := &dataFilter{}
	f.field = parts[0]
	t, ok := mapping[f.field]
	if !ok {
		internal.Info(fmt.Sprintf("filter field unknown: %s", f.field))
		return nil
	}
	f.op = stringToOp(parts[1])
	if f.op == internal.InvalidOp {
		internal.Info("filter op invalid")
		return nil
	}
	f.fxn = t
	switch t {
	case IntConv:
		i, e := strconv.Atoi(val)
		if e != nil {
			internal.Info("filter is not an int")
			return nil
		}
		f.intVal = i
	case Int64Conv:
		i, e := strconv.ParseInt(val, 10, 64)
		if e != nil {
			internal.Info("filter is not an int64")
			return nil
		}
		f.int64Val = i
	case Float64Conv:
		i, e := strconv.ParseFloat(val, 64)
		if e != nil {
			internal.Info("filter is not a float64")
		}
		f.float64Val = i
	case StrConv:
		if f.op == internal.Equals || f.op == internal.NEquals {
			f.strVal = val
		} else {
			internal.Info("filter string op is invalid")
			return nil
		}
	default:
		internal.Info("unknown filter type")
		return nil
	}
	return f
}

func timeFilter(op, value string, mapping map[string]internal.TypeConv) *dataFilter {
	return parseFilter(fmt.Sprintf("%s%s%s%s%s", internal.TSKey, filterDelimiter, op, filterDelimiter, value), mapping)
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

func (t *TagAdder) add(first bool, j map[string]json.RawMessage) {
	if first {
		t.tracked = make(map[string]*tagMeta)
	}
	o, ok := getSubField(internal.FieldKey, j)
	if !ok {
		return
	}
	o, ok = getSubField(internal.TagKey, o)
	if !ok {
		return
	}
	v, ok := o[internal.NotJSON]
	if !ok {
		return
	}
	tsRaw, ok := j[internal.TSKey]
	if !ok {
		return
	}
	dtRaw, ok := j[internal.DTKey]
	if !ok {
		return
	}
	d, ok := internal.JSONstring(dtRaw)
	if !ok {
		return
	}
	i, ok := internal.JSONint64(tsRaw)
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

func (t *TagAdder) done(ctx *Context, w io.Writer, limit bool) {
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

// NewDataWriter inits a new data writer for use
func NewDataWriter(w io.Writer, h onHeaders) *DataWriter {
	o := &DataWriter{}
	o.write = w != nil
	o.writer = w
	o.header = h != nil
	o.headers = h
	o.limit = true
	return o
}

func (d *DataWriter) setHeaders() {
	if d.header {
		d.headers()
	}
}

func (d *DataWriter) add(b []byte) {
	if d.write {
		d.writer.Write(b)
	}
}

func (d *DataWriter) addString(s string) {
	d.add([]byte(s))
}

// Handle is how we handle data requests
func Handle(ctx *Context, req map[string][]string, h *internal.Configuration, writer *DataWriter) bool {
	dataFilters := []*dataFilter{}
	limited := 0
	if writer.limit {
		limited = ctx.Limit
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
				f := parseFilter(val, ctx.Convert)
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
			f := timeFilter(mode, p[0], ctx.Convert)
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
	stime := getDate(startDate, ctx.ScanStart)
	etime := getDate(endDate, ctx.ScanEnd)
	dirs, e := ioutil.ReadDir(ctx.Directory)
	if seek && len(dirs) > 0 {
		last := dirs[len(dirs)-1]
		dirs = []os.FileInfo{last}
	}
	if e != nil {
		internal.Errored("unable to read dir", e)
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
			p := filepath.Join(ctx.Directory, dname)
			f, e := ioutil.ReadDir(p)
			if e != nil {
				internal.Info(fmt.Sprintf("unable to read subdir: %s", dname))
				internal.Errored("reading subdir failed", e)
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
							internal.Info(fmt.Sprintf("unable to unmarshal obj: %s (%s)", p, d.field))
							internal.Errored("unmarshal error", err)
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

func newWebDataWriter(w http.ResponseWriter) *DataWriter {
	return NewDataWriter(w, func() {
		writeSuccess(w)
	})
}

func (d *DataWriter) addObject(first bool, o map[string]json.RawMessage) {
	if d.object {
		d.objects.add(first, o)
	}
}

func (d *DataWriter) closeObjects(ctx *Context, limited bool) {
	if d.object {
		d.objects.done(ctx, d.writer, limited)
	}
}

func webRequest(ctx *Context, h *internal.Configuration, w http.ResponseWriter, r *http.Request, d *DataWriter) {
	success := Handle(ctx, r.URL.Query(), h, d)
	if !success {
		w.WriteHeader(http.StatusBadRequest)
	}
}

// ObjectWriter appends an object to the writer/output data
func (d *DataWriter) ObjectWriter(adder objectAdder) {
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

func apiMeta(ctx *Context, started string) []byte {
	return []byte(fmt.Sprintf("%s {\"started\": \"%s\"} %s", ctx.metaHeader, started, ctx.metaFooter))
}

// SetMeta indicates metadata for the context run
func (ctx *Context) SetMeta(version, host string) {
	ctx.metaHeader = "{\"meta\": {\"spec\": \"" + spec + "\", \"api\": \"" + version + "\", \"server\": \"" + host + "\"}, \"data\": ["
	ctx.metaFooter = "]}"
	ctx.byteHeader = []byte(ctx.metaHeader)
	ctx.byteFooter = []byte(ctx.metaFooter)
}

// Run runs the API listener
func Run(vers string) {
	conf := internal.Startup(vers)
	dir := conf.Global.Output
	bind := conf.API.Bind
	limit := conf.API.Limit
	ctx := &Context{}
	ctx.Limit = limit
	ctx.Directory = dir
	ctx.Convert = DefaultConverters()
	ctx.ScanStart = time.Duration(conf.API.StartScan) * 24 * time.Hour
	ctx.ScanEnd = time.Duration(conf.API.EndScan) * 24 * time.Hour
	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}
	ctx.SetMeta(vers, host)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		d := newWebDataWriter(w)
		webRequest(ctx, conf, w, r, d)
	})
	apiBytes := apiMeta(ctx, time.Now().Format("2006-01-02T15:04:05"))
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w)
		w.Write(apiBytes)
	})
	http.HandleFunc("/tags", func(w http.ResponseWriter, r *http.Request) {
		obj := newWebDataWriter(w)
		obj.limit = false
		obj.ObjectWriter(&TagAdder{})
		webRequest(ctx, conf, w, r, obj)
	})
	err = http.ListenAndServe(bind, nil)
	if err != nil {
		internal.Errored("unable to do http serve", err)
		panic("unable to host")
	}
}

func loadFile(path string, h *internal.Configuration) (map[string]json.RawMessage, []byte) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		internal.Info(fmt.Sprintf("error reading file: %s", path))
		internal.Errored("unable to read file", err)
		return nil, nil
	}
	var obj map[string]json.RawMessage
	err = json.Unmarshal(b, &obj)
	if err != nil {
		internal.Info(fmt.Sprintf("unable to marshal object: %s", path))
		internal.Errored("unable to parse json", err)
		return nil, nil
	}
	if h.API.Handlers.Enable {
		if !h.API.Handlers.Dump {
			_, ok := obj[internal.DumpKey]
			if ok {
				delete(obj, internal.DumpKey)
			}
		}
		v, ok := obj[internal.FieldKey]
		if ok {
			var fields map[string]*internal.Entry
			err = json.Unmarshal(v, &fields)
			if err == nil {
				rewrite := messages.HandleEntries(fields, h)
				r, err := json.Marshal(rewrite)
				if err == nil {
					obj[internal.FieldKey] = r
					b, _ = json.Marshal(obj)
				}
			}
		}
	}
	return obj, b
}
