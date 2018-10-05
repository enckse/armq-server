#!/bin/bash
DS=dataset/
BIN=bin/
CMD=../cmd/
rm -rf $BIN
mkdir -p $BIN
TODAY=$(date +%Y-%m-%d)
for f in $(ls $DS); do
    n=$(echo "$f" | cut -d "." -f 2-)
    cp $DS$f $BIN$TODAY.$n
done
for f in $(ls $CMD); do
    cat $CMD$f | sed "s/func main/func noop/g" > $BIN/$f
done
