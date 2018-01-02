#!/usr/bin/python
"""
Receive data from ARMA extensions and store.

Accepts BROADCAST messages only.
"""
import threading
import queue
import redis
import time
import logging
import socket
import redis
import time
import argparse
import json
import html
from flask import Flask, jsonify, url_for, current_app
import subprocess


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
_JSON = "json"
_COUNT = "count"

# payload information
_TAG_INDEX = 1

# flask
_ENDPOINTS = "/armq/"
_P_AFTER = "<after>"
_P_START = "<start>"
_P_END = "<end>"
_P_BUCKET = "<bucket>"
_P_TAG = "<tag>"
_P_START_END = _P_START + "/" + _P_END
_P_TAG_DATA_BUCKET = "tag/" + _P_TAG + "/data/" + _P_BUCKET
app = Flask(__name__)
_API_PAR = {}
_API_PAR[_P_AFTER] = "after epoch time"
_API_PAR[_P_START] = "starting from"
_API_PAR[_P_END] = "ending at"
_API_PAR[_P_TAG] = "tag identifier"
_API_PAR[_P_BUCKET] = "bucket"


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


def _get_object(obj):
    if obj is not None and len(obj) > 0:
        log.debug(obj)
        return "".join([x.decode("utf-8").strip() for x in obj])
    else:
        return None


def interrogate(q):
    """Interrogate commands."""
    global lock
    global RUNNING
    run = True
    tracked = []
    while run:
        try:
            obj_str = _get_object(q.get())
            if obj_str is not None:
                log.debug(obj_str)
                if _DELIMITER in obj_str:
                    parts = obj_str.split(_DELIMITER)
                    if len(parts) >= 1:
                        tag = parts[1]
                        if tag not in tracked:
                            log.info("new tag detected ({})".format(tag))
                            subprocess.call(["/usr/bin/didumumble-signal"])
                            tracked.append(tag)
        except Exception as e:
            log.warn("interrogation error")
            log.warn(e)
        with lock:
            run = RUNNING


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
            obj_str = _get_object(q.get())
            if obj_str is not None:
                bucket = int(time.time() / bucketing)
                log.debug(bucket)
                log.debug(obj_str)
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
                        choices=[SERVER, ADMIN, API])
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
    i = queue.Queue()
    queued = [q, i]
    thread = threading.Thread(target=process, args=(q,
                                                    _REDIS_HOST,
                                                    _REDIS_PORT,
                                                    _REDIS_BUCKETS))
    thread.daemon = True
    thread.start()
    read = threading.Thread(target=interrogate, args=(i,))
    read.daemon = True
    read.start()
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
            for qs in queued:
                qs.put(rcv)
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


@app.route(_ENDPOINTS + "buckets")
def get_buckets():
    """Get all buckets."""
    return _get_available_buckets(None)


@app.route(_ENDPOINTS + "buckets/" + _P_AFTER)
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


@app.route(_ENDPOINTS + _P_BUCKET + "/metadata")
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


@app.route(_ENDPOINTS + _P_TAG_DATA_BUCKET + "/json/" + _P_START_END)
def get_tag_data_by_bucket_json(tag, bucket, start, end):
    """Get tags by bucket (data)."""
    return _get_tag_data_by_bucket(tag, bucket, True, start, end)


@app.route(_ENDPOINTS + _P_TAG_DATA_BUCKET + "/raw/" + _P_START_END)
def get_tag_data_by_bucket_raw(tag, bucket, start, end):
    """Get tags by bucket (data) without auto-checking for JSON."""
    return _get_tag_data_by_bucket(tag, bucket, False, start, end)


def _get_tag_data_by_bucket(tag, bucket, auto_json, start, end):
    """Get tag data by buckets with auto-json conversion on/off."""
    r = _redis()
    data = _new_response()
    b = _get_one_bucket(r, bucket)
    data[_PAYLOAD] = []
    start_idx = int(start)
    end_idx = int(end)
    data_count = 0
    if b is not None and start <= end:
        is_next = False
        for scan in sorted(list(_get_buckets(r))):
            if is_next:
                _new_meta(data, _NEXT, scan)
                break
            if scan == b:
                is_next = True
        for entry in r.lrange(bucket, start_idx, end_idx):
            try:
                entries = _disect(entry)
                entry_tag = entries[_TAG_INDEX]
                if not _is_tag(entry_tag):
                    continue
                if entry_tag != tag:
                    continue
                jsoned = []
                is_json = []
                idx = 0
                for item in _disect(entry):
                    append = item
                    if auto_json:
                        clean = item.strip()
                        if clean.startswith("{") or clean.startswith("["):
                            try:
                                append = json.loads(clean)
                                is_json.append(idx)
                            except Exception as e:
                                _mark_error(data, "invalid json")
                                log.warn(e)
                    jsoned.append(append)
                    idx += 1
                if auto_json:
                    jsoned.append({_JSON: is_json})
                data[_PAYLOAD].append(jsoned)
                data_count += 1
            except Exception as e:
                _mark_error(data,
                            "parse error {}".format(entry.decode("utf-8")))
                log.warn(e)
    _new_meta(data, _COUNT, data_count)
    return jsonify(data)


def _api_doc(add_to, tag, text):
    """Simple api doc string output."""
    html_string = "<{}>{}</{}>".format(tag, html.escape(text), tag)
    add_to.append(html_string)


@app.route("/")
def index():
    """Index page."""
    segments = []
    _api_doc(segments, "h1", "armq api")
    _api_doc(segments, "div", "api to query the armq collections")
    rules = {}
    for r in current_app.url_map.iter_rules():
        if r.rule == "/" or r.rule.startswith("/static/"):
            continue
        rules[r.rule.replace("<", "").replace(">", "")] = r
    for rule in sorted(rules.keys()):
        r = rules[rule]
        desc = current_app.view_functions.get(r.endpoint).__doc__
        if desc is None or len(desc) == 0:
            desc = "no description"
        segments.append("<hr />")
        _api_doc(segments, "div", desc)
        _api_doc(segments, "pre", r.rule)
        has_params = False
        for p in r.rule.split("/"):
            if p.startswith("<") and p.endswith(">"):
                if not has_params:
                    _api_doc(segments, "div", "parameters")
                    has_params = True
                param = "no parameter description"
                if p in _API_PAR:
                    param = _API_PAR[p]
                _api_doc(segments, "small", "{} ({})".format(p, param))
                segments.append("<br />")
    segment_html = "".join(segments)
    return "<html><body>{}</body></html>".format(segment_html)


@app.route(_ENDPOINTS + "tags")
def get_tags():
    """Get all tags."""
    return _get_available_tags(None)


@app.route(_ENDPOINTS + "tags/" + _P_AFTER)
def get_tags_after(after):
    """Get tags after a epoch time."""
    return _get_available_tags(int(after))


def _get_available_tags(epoch):
    """Get tags available (after an epoch time?)."""
    r = _redis()
    data = _new_response()
    data[_PAYLOAD] = {}

    def _disect_tag(key, element):
        """disect a tag."""
        obj = r.lindex(key, element)
        vals = _disect(obj)
        if len(vals) > 1:
            return _get_tag(obj)
        return None

    def _new_item(tag, key):
        """Create a new tag entry."""
        if tag is not None and tag not in data[_PAYLOAD]:
            sliced = _get_epoch_as_dt(key * _REDIS_BUCKETS)
            data[_PAYLOAD][tag] = {"start": key, "dt": sliced}

    buckets = sorted(list(_get_buckets(r, after=epoch)))
    for k in buckets:
        val = None
        try:
            val = int(k)
        except ValueError:
            continue
        try:
            first_tag = _disect_tag(k, 0)
            last_tag = _disect_tag(k, -1)
            for t in [first_tag, last_tag]:
                _new_item(t, val)
            if first_tag != last_tag:
                # must scan...
                for i in r.lrange(k, 0, -1):
                    i_tag = _get_tag(i)
                    _new_item(i_tag, val)
        except Exception as e:
            _mark_error(data, "unable to get tag {}".format(k))
            log.warn(e)
            continue
    return jsonify(data)


if __name__ == '__main__':
    main()
