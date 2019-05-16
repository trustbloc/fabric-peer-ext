#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -e


DOCKER_CMD=docker


${DOCKER_CMD} run --rm -v $(pwd):/goapp -e RUN=1 -e REPO=github.com/trustbloc/bloc-node golangci/build-runner goenvbuild
