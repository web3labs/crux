#!/bin/bash
set -eu -o pipefail

# make/install quorum
git clone https://github.com/ConsenSys/quorum.git
pushd quorum >/dev/null
git checkout tags/v2.0.3-grpc
make all
cp build/bin/geth /usr/local/bin
cp build/bin/bootnode /usr/local/bin
rm -r build
popd >/dev/null

# make/install crux
git clone https://github.com/blk-io/crux.git
cd crux
git checkout tags/v1.0.3
make setup && make
cp bin/crux /usr/local/bin
rm -r bin
