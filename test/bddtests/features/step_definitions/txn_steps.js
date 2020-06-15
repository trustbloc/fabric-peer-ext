/*
    Copyright SecureKey Technologies Inc. All Rights Reserved.

    SPDX-License-Identifier: Apache-2.0
*/

var {defineSupportCode} = require('cucumber');

defineSupportCode(function ({And, But, Given, Then, When}) {
    And(/^txn service is invoked on channel "([^"]*)" with chaincode "([^"]*)" with args "([^"]*)" on peers "([^"]*)"$/, function (arg1, arg2, arg3, arg4, callback) {
        callback.pending();
    });
    And(/^txn service is invoked on channel "([^"]*)" with chaincode "([^"]*)" with args "([^"]*)" on peers "([^"]*)" then the error response should contain "([^"]*)"$/, function (arg1, arg2, arg3, arg4, arg5, callback) {
        callback.pending();
    });
    And(/^variable "([^"]*)" is computed from the identity "([^"]*)" and nonce "([^"]*)"$/, function (arg1, arg2, arg3, callback) {
        callback.pending();
    });
    And(/^variable "([^"]*)" is assigned the base64 URL-encoded value "([^"]*)"$/, function (arg1, arg2, callback) {
        callback.pending();
    });
    And(/^the endorsement response is saved to variable "([^"]*)"$/, function (arg1, callback) {
        callback.pending();
    });
    And(/^a signed proposal is created for chaincode "([^"]*)" with args "([^"]*)" with org "([^"]*)" on channel "([^"]*)" and is saved to variable "([^"]*)"$/, function (arg1, arg2, arg3, arg4, arg5, callback) {
        callback.pending();
    });
    And(/^a signed proposal with an invalid signature is created for chaincode "([^"]*)" with args "([^"]*)" with org "([^"]*)" on channel "([^"]*)" and is saved to variable "([^"]*)"$/, function (arg1, arg2, arg3, arg4, arg5, callback) {
        callback.pending();
    });
    And(/^the signed proposal "([^"]*)" is sent to peers "([^"]*)" and the responses are saved to variable "([^"]*)"$/, function (arg1, arg2, arg3, callback) {
        callback.pending();
    });
    And(/^the signed proposal "([^"]*)" is sent to peers "([^"]*)" and the invalid responses are saved to variable "([^"]*)"$/, function (arg1, arg2, arg3, callback) {
        callback.pending();
    });
});