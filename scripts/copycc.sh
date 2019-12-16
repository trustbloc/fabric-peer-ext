#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

declare -a arr=(
   "e2e_cc"
)

function finish {
  rm -rf vendor
}
trap finish EXIT

go mod vendor


echo "Copy cc..."
for i in "${arr[@]}"
do

mkdir -p ./.build/cc/src/github.com/trustbloc/fabric-peer-ext/test/chaincode/"${i}"
cp -r test/bddtests/fixtures/testdata/chaincode/"${i}"/* ./.build/cc/src/github.com/trustbloc/fabric-peer-ext/test/chaincode/"${i}"/
find ./vendor/ -type f -name go.mod -delete
find ./vendor/ -type f -name go.sum -delete
find ./vendor ! -name '*_test.go' | cpio -pdm ./.build/cc/src/github.com/trustbloc/fabric-peer-ext/test/chaincode/"${i}"

done
