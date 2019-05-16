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
# fabric-mod (May 16, 2019)
git checkout 60d0b3b2375b83dbf5e8f0f3c4e83caadbffc331

# Rewrite viper import to allow plugins to load different version of viper
sed 's/\github.com\/spf13\/viper.*/github.com\/spf13\/oldviper v0.0.0/g' -i fabric-peer-ext/mod/peer/go.mod
sed -e "\$areplace github.com/spf13/oldviper => github.com/spf13/viper v0.0.0-20150908122457-1967d93db724" -i fabric-peer-ext/mod/peer/go.mod
sed 's/\github.com\/spf13\/viper.*/github.com\/spf13\/oldviper v0.0.0/g' -i fabric-peer-ext/go.mod
sed -e "\$areplace github.com/spf13/oldviper => github.com/spf13/viper v0.0.0-20150908122457-1967d93db724" -i fabric-peer-ext/go.mod
sed 's/\github.com\/spf13\/viper.*/github.com\/spf13\/oldviper v0.0.0/g' -i go.mod
sed -e "\$areplace github.com/spf13/oldviper => github.com/spf13/viper v0.0.0-20150908122457-1967d93db724" -i go.mod
find . -type f -name "*.go" -print0 | xargs -0 sed -i "s/github.com\/spf13\/viper/github.com\/spf13\/oldviper/g"

# apply custom modules
sed 's/\.\/extensions/.\/fabric-peer-ext\/mod\/peer/g' -i go.mod
sed  -e "\$arequire  github.com/trustbloc/fabric-peer-ext v0.0.0" -i go.mod
sed  -e "\$areplace  github.com/trustbloc/fabric-peer-ext => .\/fabric-peer-ext" -i go.mod
