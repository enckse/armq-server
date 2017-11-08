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
from systemd.journal import JournalHandler

RUNNING = True
lock = threading.RLock()

log = logging.getLogger('armq')
log.addHandler(JournalHandler(SYSLOG_IDENTIFIER='armq'))
log.setLevel(logging.INFO)

# cmds
SNAP = "snapshot"
STOP = "kill"
TEST = "test"
FLUSH = "flush"

# modes
SERVER = "server"
ADMIN = "admin"

# Constants
ACK = "ack"


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
    parser.add_argument('--rport', type=int, default=6379)
    parser.add_argument('--rserver', type=str, default='localhost')
    parser.add_argument('--bucket', type=int, default=100)
    parser.add_argument('--port', type=int, default=5000)
    parser.add_argument('--command',
                        type=str,
                        default=None,
                        choices=[FLUSH, STOP, TEST, SNAP])
    parser.add_argument('--mode',
                        type=str,
                        default=None,
                        choices=[SERVER, ADMIN])
    args = parser.parse_args()
    server_mode = True
    if args.mode:
        server_mode = args.mode == SERVER
        if args.mode == ADMIN:
            admin(args)
    if server_mode:
        server(args)


def server(args):
    """Host the receiving server."""
    global lock
    global RUNNING
    q = queue.Queue()
    thread = threading.Thread(target=process, args=(q,
                                                    args.rserver,
                                                    args.rport,
                                                    args.bucket))
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

if __name__ == '__main__':
    main()
