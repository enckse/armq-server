#!/bin/bash
function _kill()
{
    echo "killing server"
    _msg "kill" $1 $2
    end=1
    while [ $end -eq 1 ]; do
        p=$(pgrep -f "python armq_server.py")
        if [ -z "$p" ]; then
            end=0
        else
            _msg "end" $1 $2
        fi
    done
}

_msg() {
    echo "$1" | nc $2 $3 &
}

function _flush()
{
    _msg "flush" $1 $2
}

function _test()
{
    _msg "test" $1 $2
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
