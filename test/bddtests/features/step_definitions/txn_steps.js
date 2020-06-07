/*
    Copyright SecureKey Technologies Inc. All Rights Reserved.

    SPDX-License-Identifier: Apache-2.0
*/

var {defineSupportCode} = require('cucumber');

defineSupportCode(function ({And, But, Given, Then, When}) {
    And(/^txn service is invoked on channel "([^"]*)" with chaincode "([^"]*)" with args "([^"]*)" on peers "([^"]*)"$/, function (arg1, arg2, arg3, arg4, callback) {
        callback.pending();
    });
});