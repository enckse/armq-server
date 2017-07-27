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

RUNNING = True
lock = threading.RLock()


def process(q, host, port, bucketing):
    """process data."""
    global lock
    global RUNNING
    run = True
    r = redis.StrictRedis(host=host, port=port, db=0)
    print("connected to redis")
    count = 0
    while run:
        try:
            obj = q.get()
            if obj is not None:
                bucket = int(time.time() / bucketing)
                print(bucket)
                r.rpush(str(bucket), obj.decode("utf-8"))
                if count > 100:
                    r.save()
                    count = 0
                count += 1
        except Exception as e:
            print('processing error')
            print(e)
        with lock:
            run = RUNNING
    try:
        r.save()
    except Exception as e:
        print('exit error')
        print(e)
    print('background processing completed')


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
        print('killed')
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
            print('socket error')
            print(e)
        with lock:
            run = RUNNING

if __name__ == '__main__':
    main()
