#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -e


GOLANGCI_LINT_CMD=golangci-lint

PWD=`pwd`
echo "Running golangci-lint :: pwd" $PWD

cd ./extensions
echo "extensions project path" `pwd`

$GOLANGCI_LINT_CMD run -c "$GOPATH/src/github.com/trustbloc/fabric-peer-ext/golangci.yml"

cd $PWD

echo "golangci-lint finished successfully :: pwd" $PWD
