#!/bin/bash
set -u
set -e

DDIR="qdata/c"
mkdir -p $DDIR
mkdir -p qdata/logs
echo $CRUX_PUB >> "$DDIR/tm.pub"
echo $CRUX_PRIV >> "$DDIR/tm.key"
rm -f "$DDIR/tm.ipc"
CMD="crux --url=http://$OWN_URL:$PORT/ --port=$PORT --workdir=$DDIR --socket=tm.ipc --publickeys=tm.pub --privatekeys=tm.key --othernodes=$OTHER_NODES --verbosity=3"
$CMD >> "qdata/logs/crux.log" 2>&1 &

DOWN=true
while $DOWN; do
    sleep 0.1
    DOWN=false
	if [ ! -S "qdata/c/tm.ipc" ]; then
            DOWN=true
	fi
done
