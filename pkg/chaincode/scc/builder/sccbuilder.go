/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package builder

import (
	"reflect"
	"sync"

	"github.com/hyperledger/fabric/core/scc"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/injectinvoker"
)

var sdSysSccType = reflect.TypeOf((*scc.SelfDescribingSysCC)(nil)).Elem()

// SCCBuilder builds a slice of SelfDescribingSysCC
type SCCBuilder struct {
	mutex    sync.Mutex
	creators []interface{}
}

// New returns a new SCCBuilder
func New() *SCCBuilder {
	return &SCCBuilder{}
}

// Add adds a new SCC creator function. The function has zero or more args
// that define dependencies and a single return value of type scc.SelfDescribingSysCC.
// Note: The args must all be interfaces.
func (b *SCCBuilder) Add(creator interface{}) *SCCBuilder {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.creators = append(b.creators, creator)
	return b
}

// Build returns a slice of SelfDescribingSysCC using the given set of providers
func (b *SCCBuilder) Build(providers ...interface{}) ([]scc.SelfDescribingSysCC, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	invoker := injectinvoker.New(providers...)
	descs := make([]scc.SelfDescribingSysCC, len(b.creators))
	for i, creator := range b.creators {
		desc, err := create(invoker, creator)
		if err != nil {
			return nil, errors.WithMessagef(err, "Error creating system chaincode [%s]", reflect.TypeOf(creator))
		}
		descs[i] = desc
	}
	return descs, nil
}

func create(invoker *injectinvoker.Invoker, fctn interface{}) (scc.SelfDescribingSysCC, error) {
	retVals, err := invoker.Invoke(fctn)
	if err != nil {
		return nil, err
	}

	// The function must return exactly one value, which is the SelfDescribingSysCC
	if len(retVals) != 1 {
		return nil, errors.New("invalid number of return values - expecting return value [scc.SelfDescribingSysCC]")
	}
	retVal := retVals[0]
	if !retVal.Type().Implements(sdSysSccType) {
		return nil, errors.New("invalid return value - expecting return value [scc.SelfDescribingSysCC]")
	}
	return retVal.Interface().(scc.SelfDescribingSysCC), nil
}
