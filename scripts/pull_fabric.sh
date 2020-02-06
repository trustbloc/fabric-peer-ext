#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

mkdir -p $GOPATH/src/github.com/hyperledger/fabric
git clone https://github.com/trustbloc/fabric-mod.git $GOPATH/src/github.com/hyperledger/fabric
cp -r . $GOPATH/src/github.com/hyperledger/fabric/fabric-peer-ext
cd $GOPATH/src/github.com/hyperledger/fabric
git config advice.detachedHead false
# fabric-mod (Feb 7, 2020)
git checkout d251d82040285b5a0bc104735c049cebd15cf622

# apply custom modules
sed 's/\.\/extensions/.\/fabric-peer-ext\/mod\/peer/g' -i go.mod
sed  -e "\$arequire  github.com/trustbloc/fabric-peer-ext v0.0.0" -i go.mod
sed  -e "\$areplace  github.com/trustbloc/fabric-peer-ext => .\/fabric-peer-ext" -i go.mod
