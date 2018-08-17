#!/bin/bash

./crux_start.sh

if $CLIENT; then
    cd /go/src/client
    go test .
#    ./client >> "client.log" 2>&1
fi
