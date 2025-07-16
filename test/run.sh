#!/bin/sh

(cd ../yell/ && go build)

for F in `ls *.jdr`; do
    echo "*** Running " $F " ***"
    ../yell/yell $F
done
