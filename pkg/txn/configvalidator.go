/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txn

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/client"
)

// configValidator validates the transaction service configuration
type configValidator struct {
}

func newConfigValidator() *configValidator {
	return &configValidator{}
}

func (v *configValidator) Validate(kv *config.KeyValue) error {
	if kv.AppName != configApp {
		return nil
	}

	if kv.PeerID == "" {
		return errors.Errorf("expecting PeerID for %s", kv.Key)
	}

	switch kv.ComponentName {
	case generalConfigComponent:
		return v.validateTXNConfig(kv)
	case sdkConfigComponent:
		return v.validateSDK(kv)
	default:
		return errors.Errorf("unexpected component [%s] for %s", kv.ComponentName, kv)
	}
}

func (v *configValidator) validateTXNConfig(kv *config.KeyValue) error {
	logger.Debugf("Validating TXN config %s", kv)

	if kv.ComponentVersion != generalConfigVersion {
		return errors.Errorf("unsupported component version [%s] for %s", kv.ComponentVersion, kv.Key)
	}

	if config.Format(strings.ToUpper(string(kv.Format))) != config.FormatJSON {
		return errors.Errorf("expecting format [%s] but got [%s] for %s", config.FormatJSON, kv.Format, kv.Key)
	}

	txnConfig := &txnConfig{}
	if err := json.Unmarshal([]byte(kv.Config), txnConfig); err != nil {
		return errors.WithMessagef(err, "error unmarshalling value for TXN config %s", kv.Key)
	}

	if txnConfig.User == "" {
		return errors.Errorf("field 'UserID' was not specified for TXN config %s", kv.Key)
	}

	return nil
}

func (v *configValidator) validateSDK(kv *config.KeyValue) error {
	logger.Debugf("Validating SDK config %s", kv)

	if kv.ComponentVersion != sdkConfigVersion {
		return errors.Errorf("unsupported component version [%s] for %s", kv.ComponentVersion, kv.Key)
	}

	if _, _, err := client.GetEndpointConfig([]byte(kv.Config), kv.Format); err != nil {
		return errors.WithMessagef(err, "invalid SDK config for %s", kv.Key)
	}

	return nil
}
