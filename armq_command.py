#!/usr/bin/python
"""Handles simple commands sent via the adc extension."""
import socket
import subprocess


def _go(command):
    """Run a command."""
    try:
        if command == "start":
            subprocess.call(["/usr/bin/didumumble-multibeep"])
        else:
            print("unknown command: {}".format(command))
    except Exception as e:
        print("could not execute command")
        print(e)


def main():
    """Main entry."""
    serversocket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    serversocket.bind(("127.0.0.1", 5001))
    serversocket.listen(5)
    while 1:
        (clientsocket, address) = serversocket.accept()
        run = True
        while run:
            recvd = clientsocket.recv(1024)
            if recvd is None or len(recvd) == 0:
                run = False
            else:
                _go(recvd.decode("utf-8"))


if __name__ == '__main__':
    main()
