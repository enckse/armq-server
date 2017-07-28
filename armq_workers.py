#!/usr/bin/python
"""
Process data from armq redis
"""
import redis
import argparse
import logging
from systemd.journal import JournalHandler

log = logging.getLogger('armqw')
log.addHandler(JournalHandler(SYSLOG_IDENTIFIER='armqw'))
ch = logging.StreamHandler()
log.addHandler(ch)
log.setLevel(logging.DEBUG)


def cache(redis_server):
    """Cache data."""



def common_worker(host, port, callback):
    """Common worker functionality."""
    run = True
    r = redis.StrictRedis(host=host, port=port, db=0)
    log.info("connected to redis")
    try:
        callback(r)
    except Exception as e:
        log.warn("callback error")
        log.warn(e)

def main():
    _CACHE = "cache"
    _RAW = "raw"
    opts = {}
    opts[_CACHE] = cache
    opts[_RAW] = raw
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', type=int, default=6379)
    parser.add_argument('--server', type=str, default='localhost')
    parser.add_argument('--mode', type=str, choices=opts.keys())
    args = parser.parse_args()
    common_worker(args.server, args.port, opts[args.mode])

if __name__ == '__main__':
    main()
