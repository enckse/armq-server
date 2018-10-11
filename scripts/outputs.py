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


class Event():
    def __init__(self, datum):
        self.type = get_raw(datum, "type")
        self.player = get_raw(datum, "playerid")
        self.simtime = get_raw(datum, "simtime")
        self.data = get_data(datum)


def get_data(obj):
    d = has_key(obj, _DATA)
    if d:
        a = has_key(d, _ARRAY)
        if a:
            return a
        return has_key(d, _OBJECT)
    return None


def delimit():
    print("==========")


def header(meta, tag):
    delimit()
    print("specification: {}".format(meta["spec"]))
    print("api: {}".format(meta["api"]))
    print("server: {}".format(meta["server"]))
    print("tag: {}".format(tag))
    delimit()


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
    warn("missing tag: {}", obj)
    return None


def parse_data(d, tag):
    for o in d:
        f = check_tag(o, tag)
        if f:
            yield Event(f)


def get_faction(e):
    k = has_key(e.data, "victim")
    if k:
        k = has_key(k, "faction")
        if k:
            return k
    warn("missing victim: {}", e.data)


def killed(events):
    factions = {}
    for e in events:
        if e.type == "unit_killed":
            f = get_faction(e)
            if f not in factions:
                factions[f] = 0
            factions[f] += 1
    delimit()
    print("kills:\n")
    for f in sorted(factions.keys()):
        v = factions[f]
        name = str(f)
        if f == 1:
            name = "blue"
        elif f == 2:
            name = "red"
        elif f == 3:
            name = "civilians"
        print("{}: {}".format(name, v))
    delimit()


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


if __name__ == "__main__":
    main()
