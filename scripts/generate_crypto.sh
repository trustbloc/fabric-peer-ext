#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

CRYPTOGEN_CMD="${CRYPTOGEN_CMD:-cryptogen}"
FIXTURES_PATH="${FIXTURES_PATH:-/opt/workspace/fabric-peer-ext/test/bddtests/fixtures/fabric}"

if [ -z "$FIXTURES_VERSION" ]; then
  echo "FIXTURES_VERSION is required"
  exit 1
fi

declare -a peerOrgs=(
    "org1.example.com"
    "org2.example.com"
)

declare -a ordererOrgs=(
    "example.com"
)

echo Clearing old crypto directory ...
rm -Rf ${FIXTURES_PATH}/crypto-config

echo Running cryptogen ...
${CRYPTOGEN_CMD} generate --config=${FIXTURES_PATH}/${FIXTURES_VERSION}/config/cryptogen.yaml --output=${FIXTURES_PATH}/crypto-config

# Remove unneeded ca MSP
for org in ${peerOrgs[@]}; do
    rm -Rf ${FIXTURES_PATH}/crypto-config/peerOrganizations/${org}/peers/ca.${org}/msp
done