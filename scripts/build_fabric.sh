#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

trap finish EXIT


function finish {
   echo "Remove tmp directory is $TMP ..."
   chmod -f -R +rw $TMP || :
   rm -Rf $TMP
}

MY_PATH="`dirname \"$0\"`"              # relative
MY_PATH="`( cd \"$MY_PATH\" && pwd )`"  # absolutized and normalized
if [ -z "$MY_PATH" ] ; then
  # error; for some reason, the path is not accessible
  # to the script (e.g. permissions re-evaled after suid)
  exit 1  # fail
fi

TMP=`mktemp -d 2>/dev/null || mktemp -d -t 'mytmpdir'`
echo "Build tmp directory is $TMP ..."

export GOPATH=$TMP

$MY_PATH/pull_fabric.sh

cd $GOPATH/src/github.com/hyperledger/fabric
PATH=$PATH:$GOPATH/bin make clean-all $FABRIC_COMMAND