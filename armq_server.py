#!/usr/bin/python
"""
Receive data from ARMA extensions and store.

Excepts BROADCAST messages only.
"""
import zmq
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
ch = logging.StreamHandler()
log.addHandler(ch)
log.setLevel(logging.DEBUG)

# cmds
FLUSH = "flush"
STOP = "kill"
TEST = "test"

# modes
SERVER = "server"
ADMIN = "admin"

# Constants
ACK = "ack"


def admin(args):
    """Administration of server."""
    if args.command == None:
        log.warn("no command set...")
        return
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    parts = args.bind.split(":")
    s.connect((parts[1].replace("//*", "localhost"), int(parts[2])))
    totalsent = 0
    msg = args.command.encode("utf-8")
    while totalsent < len(msg):
        sent = s.send(msg[totalsent:])
        if sent == 0:
            raise RuntimeError("socket connection broken")
        totalsent = totalsent + sent
    s.recv(len(ACK))
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
            if obj is not None:
                bucket = int(time.time() / bucketing)
                log.debug(bucket)
                log.debug(obj)
                obj_str = obj.decode("utf-8").strip()
                if obj_str == STOP:
                    log.info("stop request")
                    with lock:
                        RUNNING = False
                elif obj_str == FLUSH:
                    log.info("flushing")
                    r.save()
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
    global lock
    global RUNNING
    parser = argparse.ArgumentParser()
    parser.add_argument('--rport', type=int, default=6379)
    parser.add_argument('--rserver', type=str, default='localhost')
    parser.add_argument('--bucket', type=int, default=100)
    parser.add_argument('--bind', type=str, default="tcp://*:5555")
    parser.add_argument('--command',
                        type=str,
                        default=None,
                        choices=[FLUSH, STOP, TEST])
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
    context = zmq.Context()
    socket = context.socket(zmq.STREAM)
    socket.bind(args.bind)
    q = queue.Queue()
    thread = threading.Thread(target=process, args=(q,
                                                    args.rserver,
                                                    args.rport,
                                                    args.bucket))
    thread.daemon = True
    thread.start()
    run = True
    while run:
        try:
            clientid, rcv = socket.recv_multipart()
            q.put(rcv)
            socket.send_multipart([clientid, ACK.encode("utf-8")])
        except Exception as e:
            log.warn("socket error")
            log.warn(e)
        with lock:
            run = RUNNING
    log.info('done')

if __name__ == '__main__':
    main()
