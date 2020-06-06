#!/bin/bash
echo "Sending private transaction"
PRIVATE_CONFIG=qdata/c/tm.ipc geth --exec "loadScript(\"simpleContract.js\")" attach ipc:qdata/dd/geth.ipc
