package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/epiphyte/goutils"
)

type typeConv int
type opType int

const (
	maxOp                    = 5
	minOp                    = -1
	int64Conv       typeConv = 1
	strConv         typeConv = 2
	intConv         typeConv = 3
	float64Conv     typeConv = 4
	lessThan        opType   = 0
	equals          opType   = 1
	lessTE          opType   = 2
	greatThan       opType   = 3
	greatTE         opType   = 4
	nEquals         opType   = maxOp
	invalidOp       opType   = minOp
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

func prepare(dir, defs string, fields []string, limit int) *context {
	c := &context{}
	c.directory = dir
	c.limit = limit
	conf, err := goutils.LoadConfigDefaults(defs)
	if err != nil {
		goutils.Fatal("unable to read definitions config", err)
	}
	c.convert = make(map[string]typeConv)
	for _, field := range fields {
		key := fmt.Sprintf("[%s]", field)
		sect := conf.GetSection(key)
		t := sect.GetStringOrEmpty("type")
		p := sect.GetStringOrEmpty("path")
		if len(t) == 0 || len(p) == 0 {
			goutils.Fatal("type and path required", nil)
		}
		switch t {
		case "int":
			c.convert[p] = intConv
		case "int64":
			c.convert[p] = int64Conv
		case "string":
			c.convert[p] = strConv
		case "float64":
			c.convert[p] = float64Conv
		default:
			goutils.Fatal(fmt.Sprintf("%s is an unknown type", t), nil)
		}
	}
	return c
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
	return parseFilter(fmt.Sprintf("%s%s%s%s%s", tsJSON, filterDelimiter, op, filterDelimiter, value), mapping)
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

func loadFile(path string, h *handlerSettings) (map[string]json.RawMessage, []byte) {
	goutils.WriteDebug("reading", path)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		goutils.WriteWarn("error reading file", path)
		goutils.WriteError("unable to read file", err)
		return nil, nil
	}
	var obj map[string]json.RawMessage
	err = json.Unmarshal(b, &obj)
	if err != nil {
		goutils.WriteWarn("unable to marshal object", path)
		goutils.WriteError("unable to parse json", err)
		return nil, nil
	}
	if !h.allowDump {
		_, ok := obj[dumpKey]
		if ok {
			delete(obj, dumpKey)
		}
	}
	v, ok := obj[fieldKey]
	if ok {
		var fields map[string]*Entry
		err = json.Unmarshal(v, &fields)
		if err == nil {
			rewrite := []*Entry{}
			for k, v := range fields {
				v.name = k
				rewrite = append(rewrite, v)
			}
			rewrite = handleEntries(rewrite, h)
			r, err := json.Marshal(rewrite)
			if err == nil {
				obj[fieldKey] = r
			}
		}
	}
	return obj, b
}

func run(ctx *context, w http.ResponseWriter, r *http.Request, h *handlerSettings) bool {
	dataFilters := []*dataFilter{}
	limited := ctx.limit
	skip := 0
	startDate := ""
	endDate := ""
	fileRead := ""
	for k, p := range r.URL.Query() {
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"data\": ["))
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
			w.Write([]byte(","))
		}
		w.Write(b)
		has = true
		count += 1
	}
	w.Write([]byte("]}"))
	return true
}

func main() {
	conf := startup()
	dir := conf.GetStringOrDefault(outKey, dataDir)
	c := conf.GetSection("[api]")
	bind := c.GetStringOrDefault("bind", ":8080")
	limit := c.GetIntOrDefaultOnly("limit", 1000)
	fields := c.GetArrayOrEmpty("fields")
	defs := c.GetStringOrDefault("definitions", "/etc/armq.api.conf")
	goutils.WriteDebug("api ready")
	ctx := prepare(dir, defs, fields, limit)
	h := &handlerSettings{}
	h.allowEvent = true
	h.allowDump = true
	if c.GetFalse("eventHander") {
		h.allowEvent = false
	}
	if c.GetFalse("dumpHandler") {
		h.allowDump = false
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !run(ctx, w, r, h) {
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	err := http.ListenAndServe(bind, nil)
	if err != nil {
		goutils.Fatal("unable to do http serve", err)
	}
}
