#!/usr/bin/python
"""Data reading endpoints from redis."""
import redis
from flask import Flask, jsonify

# redis connection
_HOST = "127.0.0.1"
_PORT = 6379

app = Flask(__name__)

def _redis():
    return redis.StrictRedis(host=_HOST, port=_PORT, db=0)

@app.route("/armq/tags")
def get_tags():
    """Get all tags."""
    r = _redis()
    int_keys = {}
    first_keys = {}
    for k in r.keys():
        val = None
        try:
            val = int(k)
        except ValueError:
            continue
        int_keys[val] = k
        first = r.lrange(k, 0, 0)
        first_keys[val] = first
    print(first_keys)
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
