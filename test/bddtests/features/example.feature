#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@all
@example
Feature:

  @example_s1
  Scenario: example
    Given the channel "mychannel" is created and all peers have joined
    And "test" chaincode "examplecc" is installed from path "github.com/trustbloc/example_cc" to all peers
    And "test" chaincode "examplecc" is instantiated from path "github.com/trustbloc/example_cc" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy ""
    And chaincode "examplecc" is warmed up on all peers on the "mychannel" channel
