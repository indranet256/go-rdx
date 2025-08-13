#!/bin/sh

clear
(cd ../repl/ && go build)

for F in `ls *.jdr`; do
    echo "*** Running " $F " ***"
    ../repl/repl $F
done
