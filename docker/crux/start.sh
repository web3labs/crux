#!/bin/bash

echo $CRUX_PUB >> key.pub
echo $CRUX_PRIV >> key.priv
CMD="./bin/crux --url=http://$OWN_URL:$PORT/ --port=$PORT --publickeys=key.pub --privatekeys=key.priv --othernodes=$OTHER_NODES --verbosity=3"
$CMD >> "crux.log" 2>&1