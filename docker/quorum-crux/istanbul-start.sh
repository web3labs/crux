#!/bin/bash
set -u
set -e

mkdir -p qdata/logs
echo "[*] Starting Crux nodes"
./crux-start.sh

echo "[*] Starting Ethereum nodes"
set -v
ARGS="--txpool.globalslots 20000 --txpool.globalqueue 20000 --istanbul.blockperiod 1 --syncmode full --mine --rpc --rpcaddr 0.0.0.0 --rpcapi admin,db,eth,debug,miner,net,shh,txpool,personal,web3,quorum,istanbul "
PRIVATE_CONFIG=qdata/c/tm.ipc nohup geth --datadir qdata/dd $ARGS --rpcport $GETH_RPC_PORT --port $GETH_PORT --nodekeyhex $NODE_KEY --unlock 0 --password passwords.txt --verbosity=6 2>>qdata/logs/node.log &
set +v


