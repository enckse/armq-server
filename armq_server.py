#!/usr/bin/python
"""
Receive data from ARMA extensions and store.

Excepts BROADCAST messages only.
"""
import zmq
import threading
import queue
import signal
import redis
import time
import argparse
import logging
from systemd.journal import JournalHandler

RUNNING = True
lock = threading.RLock()

log = logging.getLogger('armq')
log.addHandler(JournalHandler(SYSLOG_IDENTIFIER='armq'))
ch = logging.StreamHandler()
log.addHandler(ch)
log.setLevel(logging.INFO)


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
                r.rpush(str(bucket), obj.decode("utf-8"))
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
    args = parser.parse_args()
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

    def signal_handler(signal, frame):
        """Signal handler to stop."""
        global RUNNING
        global lock
        log.info("received KILL")
        q.put(None)
        with lock:
            RUNNING = False
    signal.signal(signal.SIGUSR1, signal_handler)
    run = True
    while run:
        try:
            clientid, rcv = socket.recv_multipart()
            q.put(rcv)
            socket.send_multipart([clientid, "ack".encode("utf-8")])
        except Exception as e:
            log.warn("socket error")
            log.warn(e)
        with lock:
            run = RUNNING
    log.info('done')

if __name__ == '__main__':
    main()
