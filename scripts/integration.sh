#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

declare -a tests=(
   "e2e"
   "off_ledger,gossip_appdata"
   "transient_data"
   "ledger_config"
   "txn"
   "in_proc_ucc"
)

PWD=`pwd`
cd test/bddtests

totalAgents=${SYSTEM_TOTALJOBSINPHASE:-0}   # standard VSTS variables available using parallel execution; total number of parallel jobs running
agentNumber=${SYSTEM_JOBPOSITIONINPHASE:-0} # current job position
testCount=${#tests[@]}

# below conditions are used if parallel pipeline is not used. i.e. pipeline is running with single agent (no parallel configuration)
if [ "$agentNumber" -eq 0 ]; then agentNumber=1; fi

if [ "$totalAgents" -eq 0 ]; then
  echo "***** Running e2e test in non-clustered mode..."
  DOCKER_COMPOSE_FILE=docker-compose-nocluster.yml go test -count=1 -v -cover . -p 1 -timeout=20m -run e2e

  echo "***** Running fabric-peer-ext integration tests in clustered mode..."
  go test -count=1 -v -cover . -p 1 -timeout=20m
else
  if [ "$agentNumber" -gt $totalAgents ]; then
    echo "No more tests to run"
  else
    for ((i = "$agentNumber"; i <= "$testCount"; )); do
      testToRun=("${tests[$i - 1]}")
      if [ "$testToRun" = "e2e" ]; then
        echo "***** Running the following test in non-clustered mode: $testToRun"
        DOCKER_COMPOSE_FILE=docker-compose-nocluster.yml go test -count=1 -v -cover . -p 1 -timeout=20m -run $testToRun
      else
        echo "***** Running the following test in clustered mode: $testToRun"
        go test -count=1 -v -cover . -p 1 -timeout=20m -run $testToRun
      fi
      i=$((${i} + ${totalAgents}))
    done
    mv ./docker-compose.log "./docker-compose-$agentNumber.log"
  fi
fi

cd $PWD
