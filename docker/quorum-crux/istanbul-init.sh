#!/bin/bash
set -u
set -e

echo "[*] Cleaning up temporary data directories"
rm -rf qdata
mkdir -p qdata/logs

echo "[*] Configuring node"
mkdir -p qdata/dd/{keystore,geth}
echo $PERMISSIONED_NODES >> qdata/dd/static-nodes.json
echo $PERMISSIONED_NODES >> qdata/dd/permissioned-nodes.json
echo $GETH_KEY >> qdata/dd/keystore/key
geth --datadir qdata/dd init istanbul-genesis.json
