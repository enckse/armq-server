#!/usr/bin/python
import sys
import json
import os

_FIELDS = "fields"
_TAG = "tag"
_RAW = "raw"
_DATA = "data"
_ARRAY = "array"
_OBJECT = "object"
_OFFSET = "  "


class Event():
    def __init__(self, datum, datetime):
        self.type = get_raw(datum, "type")
        self.player = get_raw(datum, "playerid")
        self.simtime = get_raw(datum, "simtime")
        self.data = get_data(datum)
        self.datetime = datetime
        if datetime is None:
            self.datetime = "unknown time"


def get_data(obj):
    d = has_key(obj, _DATA)
    if d:
        a = has_key(d, _ARRAY)
        if a:
            return a
        return has_key(d, _OBJECT)
    return None


def delimit(title):
    print("")
    print(title)
    print("===")


def header(meta, tag):
    delimit("server information")
    headers = ["spec", "api", "server"]
    vals = {}
    for h in headers:
        vals[h] = meta[h]
    vals["tag"] = tag
    for v in vals.keys():
        print("{}{:10s} {}".format(_OFFSET, v, vals[v]))


def has_key(obj, key):
    if key in obj:
        return obj[key]
    return None


def get_raw(obj, key):
    o = has_key(obj, key)
    if o:
        return has_key(o, _RAW)
    return None


def die(message):
    print(message)
    exit(1)


def warn(message, obj):
    print("WARN: {}: {}".format(message, obj))


def check_tag(obj, tag):
    fields = has_key(obj, _FIELDS)
    if fields:
        raw = get_raw(fields, _TAG)
        if raw:
            if tag == raw:
                return fields
    warn("missing tag: ", obj)
    return None


def parse_data(d, tag):
    for o in d:
        f = check_tag(o, tag)
        if f:
            yield Event(f, has_key(o, "dt"))


def format_unit_type(obj):
    val = "unknown"
    u = has_key(obj, "unit")
    t = has_key(obj, "type")
    if u:
        val = u
    if t:
        val = "{} ({})".format(val, t)
    return val


def get_faction(e):
    v = has_key(e.data, "victim")
    a = has_key(e.data, "attacker")
    if v and a:
        k = has_key(v, "faction")
        if k is not None:
            return (k, format_unit_type(v), format_unit_type(a))
    warn("missing victim: ", e.data)


def killed(events):
    factions = {}
    first = True
    for e in events:
        if first:
            delimit("starting at {}".format(e.datetime))
        first = False
        if e.type == "unit_killed":
            v = get_faction(e)
            victim = v[1]
            attacker = v[2]
            if victim == attacker:
                continue
            f = v[0]
            if f == 0:
                f = "east"
            elif f == 1:
                f = "west"
            elif f == 2:
                f = "independent"
            elif f == 3:
                f = "civilian"
            else:
                f = "unknown ({})".format(v[0])
            if f not in factions:
                factions[f] = 0
            print("{}{} ({})".format(_OFFSET, e.simtime, e.datetime))
            print("{}-> {} killed {}".format(_OFFSET * 4, attacker, victim))
            factions[f] += 1
    delimit("killed")
    keys = ["side", "---"] + sorted(factions.keys())
    factions["side"] = "killed"
    factions["---"] = "---"
    for f in keys:
        v = factions[f]
        print("{}{:15s} {}".format(_OFFSET, f, v))


def main():
    """main entry."""
    if len(sys.argv) < 2:
        die("invalid inputs, no tag?")
    tag = sys.argv[1]
    j = json.loads(sys.stdin.read())
    m = has_key(j, "meta")
    if not m:
        die("invalid metadata")
    header(m, tag)
    d = has_key(j, "data")
    if not d:
        die("invalid data")
    e = list(parse_data(d, tag))
    killed(e)
    print("")


if __name__ == "__main__":
    main()
