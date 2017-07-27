#!/bin/bash
function _kill()
{
    echo "killing server"
    pid=$(pgrep -f "python armq_server.py")
    for p in $pid; do
		kill -10 $p
		echo "kill" | nc localhost 5555
    done
}

function _help()
{
    grep "^function" $0 | sed "s/^function \_//g;s/()//g"
}

if [ -z "$1" ]; then
    echo "command required"
    _help
else
    _help | grep -q "^$1$"
    if [ $? -eq 0 ]; then
        _$1
    else
        echo "invalid command"
        _help
    fi
fi
