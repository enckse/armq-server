#!/usr/bin/python
"""Process data from armq redis to other formats/functions/etc."""
import redis
import argparse
import logging
import os
from systemd.journal import JournalHandler

log = logging.getLogger('armqw')
log.addHandler(JournalHandler(SYSLOG_IDENTIFIER='armqw'))
log.setLevel(logging.DEBUG)


class Request(object):
    """Request for data from worker."""

    def __init__(self):
        """Init a request object."""
        self.bucket = None
        self.server = None
        self.since = None
        self.working = None


def _get_data(request, decode=False):
    """Get data by key, since value."""
    if decode:
        def _from_bytes(obj):
            return obj.decode("utf-8")
    else:
        def _from_bytes(obj):
            return obj
    decoding = _from_bytes
    for item in request.server.keys():
        val = None
        try:
            val = int(item) * request.bucket
        except ValueError:
            continue
        if request.since is None or val >= request.since:
            objs = [decoding(x) for x in request.server.lrange(item, 0, -1)
                    if len(x.strip()) > 0]
            if len(objs) > 0:
                yield (val, objs)


def cache(request):
    """Cache data."""
    cached = {}
    for item in _get_data(request, decode=True):
        cached[item[0]] = item[1]
    return cached


def _raw_segment(f, tag, end=""):
    """Raw segment output."""
    f.write("\n==={}==={}".format(tag, end).encode("utf-8"))


def raw(request):
    """Raw data stream to disk."""
    base_path = os.path.join(request.working, 'raw.dump')
    log.info("writing to file")
    log.info(base_path)
    with open(base_path, 'wb') as f:
        for item in _get_data(request):
            val = str(item[0]).encode("utf-8")
            for datum in item[1]:
                _raw_segment(f, "STARTS ", "\n")
                f.write(val)
                f.write(b"\n")
                f.write(datum)
                _raw_segment(f, "ENDING")


def common_worker(host, port, req, callback):
    """Common worker functionality."""
    run = True
    r = redis.StrictRedis(host=host, port=port, db=0)
    log.info("connected to redis")
    try:
        req.server = r
        callback(req)
    except Exception as e:
        log.warn("callback error")
        log.warn(e)


def main():
    """Main entry."""
    _CACHE = "cache"
    _RAW = "raw"
    opts = {}
    opts[_CACHE] = cache
    opts[_RAW] = raw
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', type=int, default=6379)
    parser.add_argument('--server', type=str, default='localhost')
    parser.add_argument('--mode', type=str, required=True, choices=opts.keys())
    parser.add_argument('--since', type=int, default=None)
    parser.add_argument('--bucket', type=int, default=100)
    parser.add_argument('--workdir', type=str, default=None)
    args = parser.parse_args()
    req = Request()
    req.bucket = args.bucket
    req.since = args.since
    req.working = ''
    if args.workdir:
        req.working = args.workdir 
    common_worker(args.server, args.port, req, opts[args.mode])

if __name__ == '__main__':
    main()
