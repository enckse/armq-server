#!/bin/bash
TODAY=$(date +%Y-%m-%d)
DS=dataset/
BIN=bin/
rm -rf $BIN
SET=$BIN$TODAY/
mkdir -p $BIN $SET
for f in $(ls $DS); do
    n=$(echo "$f" | cut -d "." -f 2-)
    cp $DS$f $SET$TODAY.$n
done
cp ../bin/armq-tests .
