#!/bin/bash
function _kill()
{
    echo "killing server"
    pid=$(pgrep -f "python armq_server.py")
    for p in $pid; do
		kill -10 $p
		echo "kill" | nc $1 $2
    done
}

function _flush()
{
    echo "" | nc $1 $2
}

function _test()
{
    echo "test" | nc $1 $2 &
}

function _help()
{
    grep "^function" $0 | sed "s/^function \_//g;s/()//g"
}

host=localhost
port=5555
if [ ! -z "$ARMQ_HOST_SERVER" ]; then
    host=$ARMQ_HOST_SERVER
fi
if [ ! -z "$ARMQ_PORT_SERVER" ]; then
    port=$ARMQ_PORT_SERVER
fi
if [ -z "$1" ]; then
    echo "command required"
    _help
else
    _help | grep -q "^$1$"
    if [ $? -eq 0 ]; then
        _$1 $host $port
    else
        echo "invalid command"
        _help
    fi
fi
