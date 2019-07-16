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

mkdir -p ./.build/cc/src/github.com/trustbloc/"${i}"
cp -r test/bddtests/fixtures/testdata/src/github.com/trustbloc/"${i}"/* ./.build/cc/src/github.com/trustbloc/"${i}"/
find ./vendor ! -name '*_test.go' | cpio -pdm ./.build/cc/src/github.com/trustbloc/"${i}"

done
