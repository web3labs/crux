#!/bin/bash
set -u
set -e

DDIR="/go/src/crux/"
rm -rf $DDIR/crux.ipc
echo $CRUX_PUB >> key.pub
echo $CRUX_PRIV >> key.priv
CMD="./bin/crux --url=http://$OWN_URL:$PORT/ --port=$PORT --workdir=$DDIR --publickeys=key.pub --privatekeys=key.priv --othernodes=$OTHER_NODES --verbosity=3"
$CMD >> "crux.log" 2>&1 &

DOWN=true
while $DOWN; do
    sleep 0.1
    DOWN=false
	if [ ! -S "/go/src/crux/crux.ipc" ]; then
            DOWN=true
	fi
done