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

app = Flask(__name__)

def _redis():
    return redis.StrictRedis(host=_HOST, port=_PORT, db=0)

def _disect(obj):
    return obj.decode("utf-8").split(_DELIMITER)

def _new_response():
    return { _PAYLOAD: None, _ERRORS: [] }

def _mark_error(response, error):
    print(error)
    response[_ERRORS].append(error)

@app.route("/armq/tags")
def get_tags():
    """Get all tags."""
    r = _redis()
    int_keys = {}
    first_keys = {}
    data = _new_response()
    for k in r.keys():
        val = None
        try:
            val = int(k)
        except ValueError:
            continue
        int_keys[val] = k
        try:
            first = r.lrange(k, 0, 0)[0]
            first_keys[val] = _disect(first)
        except Exception as e:
            _mark_error(data, "unable to get tag {}".format(k))
            print(e)
            continue
    print(first_keys)
    print("")
#    last = None
#    for k in sorted(int_keys.keys()):
#        str_key = int_keys[k]
#        first = r.lrange(str_key, 0, 0)
#        
#        objs = r.lrange(int_keys[k], 0


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
