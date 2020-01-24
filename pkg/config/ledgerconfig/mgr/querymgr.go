/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgr

import (
	"encoding/json"
	"strings"

	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/compositekey"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	state "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

// QueryManager allows you to query ledger configuration
type QueryManager struct {
	namespace         string
	retrieverProvider state.RetrieverProvider
}

// NewQueryManager returns a new configuration manager
func NewQueryManager(namespace string, p state.RetrieverProvider) *QueryManager {
	return &QueryManager{
		namespace:         namespace,
		retrieverProvider: p,
	}
}

// Query retrieves configuration based on the provided config key.
func (m *QueryManager) Query(criteria *config.Criteria) ([]*config.KeyValue, error) {
	if criteria.MspID == "" {
		return nil, errors.New("MspID is required")
	}

	// Query for keys and and out undesired records
	logger.Debugf("Searching for config by criteria [%s]", criteria)
	return m.query(criteria)
}

// Get retrieves the configuration based on the provided config key.
func (m *QueryManager) Get(key *config.Key) (*config.Value, error) {
	retriever, err := m.retrieverProvider.GetStateRetriever()
	if err != nil {
		return nil, err
	}
	defer retriever.Done()

	return m.getConfig(retriever, key)
}

func (m *QueryManager) query(criteria *config.Criteria) ([]*config.KeyValue, error) {
	if err := criteria.Validate(); err != nil {
		return nil, err
	}

	retriever, err := m.retrieverProvider.GetStateRetriever()
	if err != nil {
		return nil, err
	}
	defer retriever.Done()

	it, err := retriever.GetStateByPartialCompositeKey(m.namespace, indexMspID, []string{criteria.MspID})
	if err != nil {
		return nil, errors.WithMessagef(err, "Unexpected error retrieving message statuses with index [%s]", indexMspID)
	}
	defer func() {
		if err := it.Close(); err != nil {
			logger.Errorf("Failed to close iterator: %s", err)
		}
	}()

	var configs []*config.KeyValue
	for it.HasNext() {
		k, v, err := getNext(it,
			func(key *config.Key) (*config.Value, error) {
				return m.getConfig(retriever, key)
			},
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, &config.KeyValue{Key: k, Value: v})
	}
	return ConfigResults(configs).Filter(criteria), nil
}

func (m *QueryManager) getConfig(retriever state.StateRetriever, key *config.Key) (*config.Value, error) {
	logger.Debugf("Getting config for [%s]", key)

	strKey := MarshalKey(key)
	bytes, err := retriever.GetState(m.namespace, strKey)
	if err != nil {
		return nil, errors.WithMessagef(err, "error getting state for key [%s]", strKey)
	}
	if len(bytes) == 0 {
		logger.Debugf("Key not found [%s]", strKey)
		return nil, nil
	}

	value := &config.Value{}
	if json.Unmarshal(bytes, value) != nil {
		return nil, errors.WithMessage(err, "error unmarshalling config value")
	}

	return value, nil
}

type configRetriever func(key *config.Key) (*config.Value, error)

func getNext(it state.ResultsIterator, getConfig configRetriever) (*config.Key, *config.Value, error) {
	compositeKey, e := it.Next()
	if e != nil {
		return nil, nil, errors.WithMessage(e, "Failed to get next value from iterator")
	}

	k, err := getKeyFromCompositeKey(compositeKey)
	if err != nil {
		return nil, nil, err
	}
	v, err := getConfig(k)
	if err != nil {
		return nil, nil, err
	}
	return k, v, nil
}

func getKeyFromCompositeKey(compositeKey *queryresult.KV) (*config.Key, error) {
	_, compositeKeyParts := compositekey.Split(compositeKey.Key)
	return UnmarshalKey(compositeKeyParts[len(compositeKeyParts)-1])
}

// MarshalKey marshals the key into a string
func MarshalKey(k *config.Key) string {
	return strings.Join([]string{k.MspID, k.PeerID, k.AppName, k.AppVersion, k.ComponentName, k.ComponentVersion}, keyDivider)
}

// UnmarshalKey creates a key from the given string
func UnmarshalKey(str string) (*config.Key, error) {
	ck := &config.Key{}
	keyParts := strings.Split(str, keyDivider)
	if len(keyParts) < 6 {
		return ck, errors.Errorf("invalid config str %v", str)
	}
	ck.MspID = keyParts[0]
	ck.PeerID = keyParts[1]
	ck.AppName = keyParts[2]
	ck.AppVersion = keyParts[3]
	ck.ComponentName = keyParts[4]
	ck.ComponentVersion = keyParts[5]
	return ck, nil
}
