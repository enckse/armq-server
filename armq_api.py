#!/usr/bin/python
"""Data reading endpoints from redis."""
import redis
import time
import argparse
import json
from flask import Flask, jsonify, url_for

# redis connection
_HOST = "127.0.0.1"
_PORT = 6379
_BUCKETS = 100

# data information
_DELIMITER = "`"
_PAYLOAD = "data"
_ERRORS = "errors"
_NEXT = "next"
_AFTER_TIME = "after"
_META = "meta"

# payload information
_TAG_INDEX = 1

app = Flask(__name__)


def _redis():
    """Create redis connection."""
    return redis.StrictRedis(host=_HOST, port=_PORT, db=0)


def _disect(obj):
    """Disect data by delimiter."""
    return obj.decode("utf-8").split(_DELIMITER)


def _new_response():
    """Create a new response object (common)."""
    return {_PAYLOAD: None, _ERRORS: [], _META: {}}


def _mark_error(response, error):
    """Create an error response entry."""
    print(error)
    response[_ERRORS].append(error)


def _is_tag(string_tag):
    """Check if an entry is a tag."""
    try:
        if len(string_tag) == 4:
            check = [x for x in string_tag if x >= 'a' and x <= 'z']
            if len(check) == len(string_tag):
                return string_tag
    except Exception as e:
        return None


def _get_tag(obj):
    """Get a tag from an entry."""
    return _is_tag(_disect(obj)[_TAG_INDEX])


def _get_buckets(server, match=None):
    """Get buckets as ints."""
    for k in server.keys():
        try:
            val = int(k)
            if match is not None and val != match:
                continue
            yield val
        except ValueError:
            continue


@app.route("/armq/buckets")
def get_buckets():
    """Get all buckets."""
    return _get_available_buckets(None)


@app.route("/armq/buckets/<after>")
def get_buckets_after(after):
    """Get buckets after a specific time (epoch)."""
    return _get_available_buckets(int(after))


def _get_epoch_as_dt(epoch_time):
    """Get epoch as dt consistently (string)."""
    return time.strftime("%Y-%m-%d %H:%M:%S", time.gmtime(epoch_time))


def _new_meta(resp, key, data):
    """Metadata entry."""
    resp[_META][key] = data


def _get_available_buckets(after):
    """Get available buckets."""
    r = _redis()
    data = _new_response()
    data[_PAYLOAD] = []
    if after is not None:
        _new_meta(data, _AFTER_TIME, _get_epoch_as_dt(after))
    for b in sorted(list(_get_buckets(r))):
        epoch = b * _BUCKETS
        sliced = _get_epoch_as_dt(epoch)
        if after is not None:
            if epoch < after:
                continue
        data[_PAYLOAD].append({"bucket": b, "slice": sliced})
    return jsonify(data)


def _get_one_bucket(server, bucket):
    """Get one bucket."""
    b = list(_get_buckets(server, match=int(bucket)))
    if len(b) > 0:
        return b[0]
    else:
        return None


def _empty_params(rule):
    """Empty parameters."""
    defaults = rule.defaults if rule.defaults is not None else ()
    arguments = rule.arguments if rule.arguments is not None else ()
    return len(defaults) >= len(arguments)


@app.route("/routes")
def app_routes():
    links = []
    for rule in app.url_map.iter_rules():
        if "GET" in rule.methods and _empty_params(rule):
            url = url_for(rule.endpoint, **(rule.defaults or {}))
            links.append((url, rule.endpoint))
    data = _new_response()
    data[_PAYLOAD] = links
    return jsonify(data)


@app.route("/armq/<bucket>/metadata")
def get_bucket_metadata(bucket):
    """Get bucket metadata."""
    r = _redis()
    data = _new_response()
    b = _get_one_bucket(r, bucket)
    data[_PAYLOAD] = []
    if b is not None:
        for entry in r.lrange(bucket, 0, -1):
            tag = _get_tag(entry)
            if tag is None:
                # metadata is not tagged
                data[_PAYLOAD].append(_disect(entry))
    return jsonify(data)


@app.route("/armq/tag/<tag>/data/<bucket>")
def get_tag_data_by_bucket(tag, bucket):
    """Get tags by bucket (data)."""
    r = _redis()
    data = _new_response()
    b = _get_one_bucket(r, bucket)
    data[_PAYLOAD] = []
    if b is not None:
        is_next = False
        for scan in sorted(list(_get_buckets(r))):
            if is_next:
                _new_meta(data, _NEXT, scan)
                break
            if scan == b:
                is_next = True
        for entry in r.lrange(bucket, 0, -1):
            try:
                entries = _disect(entry)
                jsoned = []
                for item in _disect(entry):
                    append = item
                    try:
                        clean = item.strip()
                        if clean.startswith("{") or clean.startswith("["):
                            append = json.loads(clean)
                    except Exception as e:
                        pass
                    jsoned.append(append)
                    data[_PAYLOAD].append(jsoned)
            except Exception as e:
                _mark_error(data,
                            "parse error {}".format(entry.decode("utf-8")))
                print(e)
    return jsonify(data)


@app.route("/armq/tags")
def get_tags():
    """Get all tags."""
    r = _redis()
    int_keys = {}
    first_keys = {}
    data = _new_response()
    data[_PAYLOAD] = {}
    for k in _get_buckets(r):
        val = None
        try:
            val = int(k)
        except ValueError:
            continue
        int_keys[val] = k
        try:
            first = r.lrange(k, 0, 0)[0]
            disected = _disect(first)
            if len(disected) > 1:
                tag = _get_tag(first)
                if tag is None:
                    continue
                first_keys[val] = tag
        except Exception as e:
            _mark_error(data, "unable to get tag {}".format(k))
            print(e)
            continue
    last_tag = None
    interrogate = []

    def _create_tag_start(tag, key):
        """Create a new tag entry."""
        data[_PAYLOAD][tag] = {"start": key}
    for k in sorted(int_keys.keys()):
        try:
            tagged = None
            if k in first_keys:
                tagged = first_keys[k]
                if tagged == last_tag:
                    interrogate.pop()
                interrogate.append(k)
                if tagged not in data[_PAYLOAD]:
                    _create_tag_start(tagged, k)
                last_tag = tagged
        except Exception as e:
            _mark_error(data, "unable to prefetch {}".format(k))
            print(e)
    while len(interrogate) > 0:
        current = interrogate.pop()
        for item in r.lrange(int_keys[current], 0, -1):
            try:
                tag = _get_tag(item)
                if tag is None:
                    continue
                _create_tag_start(tag, current)
            except Exception as e:
                _mark_error(data, "unable to interrogate {}".format(k))
                print(e)
    return jsonify(data)


def main():
    """Main entry."""
    parser = argparse.ArgumentParser(description="armq-server API")
    parser.add_argument("--host",
                        default="0.0.0.0",
                        type=str,
                        help="host name")
    parser.add_argument("--port", default=9090, type=int, help="host port")
    args = parser.parse_args()
    app.run(host=args.host, port=args.port)


if __name__ == "__main__":
    main()
