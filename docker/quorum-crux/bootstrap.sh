#!/bin/bash
set -eu -o pipefail

# make/install quorum
git clone https://github.com/ConsenSys/quorum.git
pushd quorum >/dev/null
git checkout tags/v2.0.2-grpc
make all
cp build/bin/geth /usr/local/bin
cp build/bin/bootnode /usr/local/bin
rm -r build
popd >/dev/null

# make/install crux
git clone https://github.com/blk-io/crux.git
cd crux
git checkout tags/v1.0.0
make setup && make
cp bin/crux /usr/local/bin
rm -r bin

# install Porosity
wget -q https://github.com/jpmorganchase/quorum/releases/download/v1.2.0/porosity
mv porosity /usr/local/bin && chmod 0755 /usr/local/bin/porosity

# done!
echo "Quorum - n"
echo
echo 'The Quorum vagrant instance has been provisioned. Examples are available in ~/quorum-examples inside the instance.'
echo "Use 'vagrant ssh' to open a terminal, 'vagrant suspend' to stop the instance, and 'vagrant destroy' to remove it."