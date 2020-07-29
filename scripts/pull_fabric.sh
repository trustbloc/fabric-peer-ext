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
# fabric-mod (Jul 29, 2020)
git checkout 635d97c98d627e690364fe88c361ca613dd33bb3

declare envOS
envOS=$(uname -s)

# apply custom modules
if [ ${envOS} = 'Darwin' ]; then
/usr/bin/sed -i '' 's/\.\/extensions/.\/fabric-peer-ext\/mod\/peer/g' go.mod
/usr/bin/sed -i '' '$a\
require  github.com/trustbloc/fabric-peer-ext v0.0.0
' go.mod
/usr/bin/sed -i '' '$a\
replace  github.com/trustbloc/fabric-peer-ext => .\/fabric-peer-ext
' go.mod
else
sed 's/\.\/extensions/.\/fabric-peer-ext\/mod\/peer/g' -i go.mod
sed  -e "\$arequire  github.com/trustbloc/fabric-peer-ext v0.0.0" -i go.mod
sed  -e "\$areplace  github.com/trustbloc/fabric-peer-ext => .\/fabric-peer-ext" -i go.mod
fi

