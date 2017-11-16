#!/usr/bin/python
"""Data reading endpoints from redis."""
import redis
import argparse
from flask import Flask, jsonify

# redis connection
_HOST = "127.0.0.1"
_PORT = 6379

# data information
_DELIMITER = "`"
_PAYLOAD = "data"
_ERRORS = "errors"

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
    return {_PAYLOAD: None, _ERRORS: []}


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

@app.route("/armq/tags")
def get_tags():
    """Get all tags."""
    r = _redis()
    int_keys = {}
    first_keys = {}
    data = _new_response()
    data[_PAYLOAD] = {}
    for k in r.keys():
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
