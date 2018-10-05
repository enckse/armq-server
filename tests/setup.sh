#!/bin/bash
TODAY=$(date +%Y-%m-%d)
DS=dataset/
BIN=bin/
rm -rf $BIN
BIN=$BIN$TODAY
mkdir -p $BIN
for f in $(ls $DS); do
    n=$(echo "$f" | cut -d "." -f 2-)
    cp $DS$f $BIN$TODAY.$n
done
cp ../bin/armq-test .
