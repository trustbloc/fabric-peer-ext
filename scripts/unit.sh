#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

# Packages to exclude
PKGS=`go list github.com/trustbloc/fabric-peer-ext/pkg/... 2> /dev/null | \
                                                 grep -v /mocks | \
                                                 grep -v /api | \
                                                 grep -v /protos`
echo "Running pkg unit tests..."
go test -count=1 -tags "testing" -cover $PKGS -p 1 -timeout=10m