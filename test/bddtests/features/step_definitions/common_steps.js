/*
    Copyright SecureKey Technologies Inc. All Rights Reserved.

    SPDX-License-Identifier: Apache-2.0
*/

var {defineSupportCode} = require('cucumber');

defineSupportCode(function ({And, But, Given, Then, When}) {
    Given(/^the channel "([^"]*)" is created and all peers have joined$/, function (arg1, callback) {
        callback.pending();
    });
    And(/^collection config "([^"]*)" is defined for collection "([^"]*)" as policy="([^"]*)", requiredPeerCount=(\d+), maxPeerCount=(\d+), and blocksToLive=(\d+)$/, function (arg1, arg2, arg3, arg4, arg5, arg6, callback) {
        callback.pending();
    });
    Given(/^"([^"]*)" chaincode "([^"]*)" is installed from path "([^"]*)" to all peers$/, function (arg1, arg2, arg3, callback) {
        callback.pending();
    });
    Given(/^"([^"]*)" chaincode "([^"]*)" is instantiated from path "([^"]*)" on the "([^"]*)" channel with args "([^"]*)" with endorsement policy "([^"]*)" with collection policy "([^"]*)"$/, function (arg1, arg2, arg3, arg4, arg5, arg6, arg7, callback) {
        callback.pending();
    });
    Given(/^chaincode "([^"]*)" is warmed up on all peers on the "([^"]*)" channel$/, function (arg1, arg2, callback) {
        callback.pending();
    });
    When(/^client queries chaincode "([^"]*)" with args "([^"]*)" on the "([^"]*)" channel$/, function (arg1, arg2, arg3, callback) {
        callback.pending();
    });
    Then(/^response from "([^"]*)" to client equal value "([^"]*)"$/, function (arg1, arg2, callback) {
        callback.pending();
    });
    Given(/^we wait (\d+) seconds$/, function (arg1, callback) {
        callback.pending();
    });
    When(/^the response is saved to variable "([^"]*)"$/, function (arg1, callback) {
        callback.pending();
    });
    When(/^client queries chaincode "([^"]*)" with args "([^"]*)" on the "([^"]*)" channel then the error response should contain "([^"]*)"$/, function (arg1, arg2, arg3, arg4, callback) {
        callback.pending();
    });
    When(/^client queries chaincode "([^"]*)" with args "([^"]*)" on a single peer in the "([^"]*)" org on the "([^"]*)" channel$/, function (arg1, arg2, arg3, arg4, callback) {
        callback.pending();
    });
    When(/^client invokes chaincode "([^"]*)" with args "([^"]*)" on the "([^"]*)" channel$/, function (arg1, arg2, arg3, callback) {
        callback.pending();
    });
    Given(/^"([^"]*)" chaincode "([^"]*)" version "([^"]*)" is installed from path "([^"]*)" to all peers$/, function (arg1, arg2, arg3, arg4, callback) {
        callback.pending();
    });
    Given(/^"([^"]*)" chaincode "([^"]*)" is upgraded with version "([^"]*)" from path "([^"]*)" on the "([^"]*)" channel with args "([^"]*)" with endorsement policy "([^"]*)" with collection policy "([^"]*)"$/, function (arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, callback) {
        callback.pending();
    });
    Given(/^"([^"]*)" chaincode "([^"]*)" is upgraded with version "([^"]*)" from path "([^"]*)" on the "([^"]*)" channel with args "([^"]*)" with endorsement policy "([^"]*)" with collection policy "([^"]*)" then the error response should contain "([^"]*)"$/, function (arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, callback) {
        callback.pending();
    });
});