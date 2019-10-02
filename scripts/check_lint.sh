#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -e

echo "Running $0"

DOCKER_CMD=docker

# TODO: Revert back to golangci-lint v1.8 since the latest version panics with "out of memory".
#   Switch back to latest once 1.20 is released (1.20 includes fixes that drastically reduce memory usage)
${DOCKER_CMD} run --rm -e GOPROXY=${GOPROXY} -v $(pwd):/opt/workspace -w /opt/workspace golangci/golangci-lint:v1.18 golangci-lint run
