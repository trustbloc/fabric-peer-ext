/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgr

import (
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
)

// ConfigResults holds the results of a configuration query
type ConfigResults []*config.KeyValue

// Filter filters the results based on the given criteria
func (r ConfigResults) Filter(criteria *config.Criteria) ConfigResults {
	if criteria.PeerID == "" && criteria.AppName == "" {
		// No filter
		return r
	}
	if criteria.PeerID != "" {
		return r.filterByPeer(criteria)
	}
	return r.filterByApp(criteria)
}

func (r ConfigResults) filterByPeer(criteria *config.Criteria) ConfigResults {
	return r.and(criteria,
		mustHavePeerID,
		mayHaveAppName,
		mayHaveAppVersion,
		mayHaveComponentName,
		mayHaveComponentVersion,
	)
}

func (r ConfigResults) filterByApp(criteria *config.Criteria) ConfigResults {
	return r.and(criteria,
		mustHaveAppName,
		mayHaveAppVersion,
		mayHaveComponentName,
		mayHaveComponentVersion,
	)
}

type predicate func(v *config.KeyValue, criteria *config.Criteria) bool

type predicates []predicate

func (ps predicates) apply(kv *config.KeyValue, criteria *config.Criteria) bool {
	for _, p := range ps {
		if !p(kv, criteria) {
			return false
		}
	}
	return true
}

func (r ConfigResults) and(criteria *config.Criteria, p ...predicate) ConfigResults {
	var results ConfigResults
	for _, v := range r {
		if predicates(p).apply(v, criteria) {
			logger.Debugf("... adding [%s]", v.Key)
			results = append(results, v)
		}
	}
	return results
}

func mustHaveAppName(kv *config.KeyValue, criteria *config.Criteria) bool {
	return kv.AppName == criteria.AppName
}

func mustHavePeerID(kv *config.KeyValue, criteria *config.Criteria) bool {
	return kv.PeerID == criteria.PeerID
}

func mayHaveAppName(kv *config.KeyValue, criteria *config.Criteria) bool {
	return criteria.AppName == "" || kv.AppName == criteria.AppName
}

func mayHaveAppVersion(kv *config.KeyValue, criteria *config.Criteria) bool {
	return criteria.AppVersion == "" || kv.AppVersion == criteria.AppVersion
}

func mayHaveComponentName(kv *config.KeyValue, criteria *config.Criteria) bool {
	return criteria.ComponentName == "" || kv.ComponentName == criteria.ComponentName
}

func mayHaveComponentVersion(kv *config.KeyValue, criteria *config.Criteria) bool {
	return criteria.ComponentVersion == "" || kv.ComponentVersion == criteria.ComponentVersion
}
