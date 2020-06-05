/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package injectinvoker

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

// Builder builds a slice of resources
type Builder struct {
	mutex    sync.Mutex
	creators []interface{}
}

// NewBuilder returns a new resource builder
func NewBuilder() *Builder {
	return &Builder{}
}

// Add adds a new creator function. The function has zero or more args
// that define dependencies and a single return value.
// Note: The args must all be interfaces.
func (b *Builder) Add(creator interface{}) *Builder {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.creators = append(b.creators, creator)
	return b
}

// Build returns a slice of resources using the given set of providers
func (b *Builder) Build(providers ...interface{}) ([]interface{}, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	invoker := New(providers...)
	descs := make([]interface{}, len(b.creators))
	for i, creator := range b.creators {
		desc, err := create(invoker, creator)
		if err != nil {
			return nil, errors.WithMessagef(err, "Error creating resource [%s]", reflect.TypeOf(creator))
		}
		descs[i] = desc
	}
	return descs, nil
}

func create(invoker *Invoker, fctn interface{}) (interface{}, error) {
	retVals, err := invoker.Invoke(fctn)
	if err != nil {
		return nil, err
	}

	// The function must return exactly one value
	if len(retVals) != 1 {
		return nil, errors.New("expecting exactly one return value")
	}

	return retVals[0].Interface(), nil
}
