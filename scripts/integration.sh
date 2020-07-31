#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e


test_prefix=

if [ "${FIXTURES_VERSION}" = "v2.2" ]; then
  test_prefix="lc_"
fi

e2e_test="${test_prefix}e2e"
off_ledger_test="${test_prefix}off_ledger"
transient_data_test="${test_prefix}transient_data"
gossip_appdata_test="${test_prefix}gossip_appdata"
in_proc_ucc_test="${test_prefix}in_proc_ucc"
ledger_config_test="${test_prefix}ledger_config"
txn_test="${test_prefix}txn"

all_tests="${e2e_test},${off_ledger_test},${transient_data_test},${gossip_appdata_test},${in_proc_ucc_test},${ledger_config_test},${txn_test}"

declare -a tests=(
     "${e2e_test}"
     "${off_ledger_test},${transient_data_test},${txn_test}"
     "${in_proc_ucc_test},${ledger_config_test},${gossip_appdata_test}"
)

echo "All tests: ${all_tests}"
echo "e2e test: ${e2e_test}"

PWD=`pwd`
cd test/bddtests

totalAgents=${SYSTEM_TOTALJOBSINPHASE:-0}   # standard VSTS variables available using parallel execution; total number of parallel jobs running
agentNumber=${SYSTEM_JOBPOSITIONINPHASE:-0} # current job position
testCount=${#tests[@]}

# below conditions are used if parallel pipeline is not used. i.e. pipeline is running with single agent (no parallel configuration)
if [ "$agentNumber" -eq 0 ]; then agentNumber=1; fi

if [ "$totalAgents" -eq 0 ]; then
  echo "***** Running ${e2e_test} test in non-clustered mode..."
  DOCKER_COMPOSE_FILE=docker-compose-nocluster.yml go test -count=1 -v -cover . -p 1 -timeout=20m -run ${e2e_test}

  echo "***** Running tests ${all_tests} in clustered mode..."
  go test -count=1 -v -cover . -p 1 -timeout=20m -run ${all_tests}
else
  if [ "$agentNumber" -gt $totalAgents ]; then
    echo "No more tests to run"
  else
    for ((i = "$agentNumber"; i <= "$testCount"; )); do
      testToRun=("${tests[$i - 1]}")
      if [ "$testToRun" = "${e2e_test}" ]; then
        echo "***** Running the following test in non-clustered mode: $testToRun"
        DOCKER_COMPOSE_FILE=docker-compose-nocluster.yml go test -count=1 -v -cover . -p 1 -timeout=20m -run $testToRun
      else
        echo "***** Running the following test in clustered mode: $testToRun"
        go test -count=1 -v -cover . -p 1 -timeout=20m -run $testToRun
      fi
      i=$((${i} + ${totalAgents}))
    done
    mv ./docker-compose.log "./docker-compose-${test_prefix}$agentNumber.log"
  fi
fi

cd $PWD
