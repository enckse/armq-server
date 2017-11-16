#!/usr/bin/python
"""
Receive data from ARMA extensions and store.

Accepts BROADCAST messages only.
"""
import threading
import queue
import redis
import time
import argparse
import logging
import socket
import redis
import time
import argparse
import json
from flask import Flask, jsonify, url_for

RUNNING = True
lock = threading.RLock()
_FORMAT = '%(asctime)s - %(name)s - %(levelname)s - %(message)s'

log = logging.getLogger('armq')
ch = logging.StreamHandler()
formatter = formatter = logging.Formatter(_FORMAT)
ch.setFormatter(formatter)
ch.setLevel(logging.INFO)
log.addHandler(ch)
log.setLevel(logging.INFO)

# cmds
SNAP = "snapshot"
STOP = "kill"
TEST = "test"
FLUSH = "flush"

# modes
SERVER = "server"
ADMIN = "admin"
API = "api"

# redis connection
_REDIS_HOST = "127.0.0.1"
_REDIS_PORT = 6379
_REDIS_BUCKETS = 100

# data information
_DELIMITER = "`"
_PAYLOAD = "data"
_ERRORS = "errors"
_NEXT = "next"
_META = "meta"

# payload information
_TAG_INDEX = 1

# flask
app = Flask(__name__)


def admin(args):
    """Administration of server."""
    if args.command is None:
        log.warn("no command set...")
        return
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect(("localhost", args.port))
    totalsent = 0
    msg = args.command.encode("utf-8")
    while totalsent < len(msg):
        sent = s.send(msg[totalsent:])
        if sent == 0:
            raise RuntimeError("socket connection broken")
        totalsent = totalsent + sent
    s.shutdown(socket.SHUT_WR)


def process(q, host, port, bucketing):
    """process data."""
    global lock
    global RUNNING
    run = True
    r = redis.StrictRedis(host=host, port=port, db=0)
    log.info("connected to redis")
    count = 0
    while run:
        try:
            obj = q.get()
            if obj is not None and len(obj) > 0:
                bucket = int(time.time() / bucketing)
                log.debug(bucket)
                log.debug(obj)
                obj_str = "".join([x.decode("utf-8").strip() for x in obj])
                if obj_str == STOP:
                    log.info("stop request")
                    with lock:
                        RUNNING = False
                elif obj_str == SNAP:
                    log.info("saving")
                    r.save()
                    count = 0
                elif obj_str == FLUSH:
                    log.info('flushing')
                    r.flushall()
                    count = 0
                else:
                    r.rpush(str(bucket), obj_str)
                if count > 100:
                    r.save()
                    count = 0
                count += 1
        except Exception as e:
            log.warn("processing error")
            log.warn(e)
        with lock:
            run = RUNNING
    try:
        r.save()
    except Exception as e:
        log.warn('exit error')
        log.warn(e)
    log.info("worker done")


def main():
    """receive and background process data."""
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', type=int, default=5000)
    parser.add_argument('--command',
                        type=str,
                        default=None,
                        choices=[FLUSH, STOP, TEST, SNAP])
    parser.add_argument('--mode',
                        type=str,
                        default=None,
                        choices=[SERVER, ADMIN])
    parser.add_argument("--apihost",
                        default="0.0.0.0",
                        type=str,
                        help="api name")
    parser.add_argument("--apiport", default=9090, type=int, help="api port")
    args = parser.parse_args()
    server_mode = True
    if args.mode:
        server_mode = args.mode == SERVER
        if args.mode == ADMIN:
            admin(args)
        if args.mode == API:
            app.run(host=args.apihost, port=args.apiport)
    if server_mode:
        server(args)


def server(args):
    """Host the receiving server."""
    global lock
    global RUNNING
    q = queue.Queue()
    thread = threading.Thread(target=process, args=(q,
                                                    _REDIS_HOST,
                                                    _REDIS_PORT,
                                                    _REDIS_BUCKETS))
    thread.daemon = True
    thread.start()
    run = True
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.bind(("127.0.0.1", args.port))
    server.listen(5)
    while run:
        try:
            (clientsock, addr) = server.accept()
            reading = True
            rcv = []
            while reading:
                recvd = clientsock.recv(1024)
                if recvd is None or len(recvd) == 0:
                    reading = False
                else:
                    rcv.append(recvd)
            if len(rcv) == 0:
                continue
            q.put(rcv)
        except Exception as e:
            log.warn("socket error")
            log.warn(e)
        with lock:
            run = RUNNING
    log.info('done')


def _redis():
    """Create redis connection."""
    return redis.StrictRedis(host=_REDIS_HOST, port=_REDIS_PORT, db=0)


def _disect(obj):
    """Disect data by delimiter."""
    return obj.decode("utf-8").split(_DELIMITER)


def _new_response():
    """Create a new response object (common)."""
    return {_PAYLOAD: None, _ERRORS: [], _META: {}}


def _mark_error(response, error):
    """Create an error response entry."""
    log.error(error)
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


def _get_buckets(server, match=None, after=None):
    """Get buckets as ints."""
    for k in server.keys():
        try:
            val = int(k)
            if match is not None and val != match:
                continue
            if after is not None and val * _REDIS_BUCKETS < after:
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


def _get_available_buckets(epoch):
    """Get available buckets."""
    r = _redis()
    data = _new_response()
    data[_PAYLOAD] = []
    for b in sorted(list(_get_buckets(r, after=epoch))):
        sliced = _get_epoch_as_dt(b * _REDIS_BUCKETS)
        data[_PAYLOAD].append({"bucket": b, "slice": sliced})
    return jsonify(data)


def _get_one_bucket(server, bucket):
    """Get one bucket."""
    b = list(_get_buckets(server, match=int(bucket)))
    if len(b) > 0:
        return b[0]
    else:
        return None


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
                log.warn(e)
    return jsonify(data)


@app.route("/armq/tags")
def get_tags():
    """Get all tags."""
    return _get_available_tags(None)


@app.route("/armq/tags/<after>")
def get_tags_after(after):
    """Get tags after a epoch time."""
    return _get_available_tags(int(after))


def _get_available_tags(epoch):
    """Get tags available (after an epoch time?)."""
    r = _redis()
    int_keys = {}
    first_keys = {}
    data = _new_response()
    data[_PAYLOAD] = {}
    for k in _get_buckets(r, after=epoch):
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
            log.warn(e)
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
            log.warn(e)
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
                log.warn(e)
    return jsonify(data)


if __name__ == '__main__':
    main()
