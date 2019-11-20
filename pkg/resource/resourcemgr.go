/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resource

import (
	"reflect"
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/injectinvoker"
)

var logger = flogging.MustGetLogger("ext_resource")

// Mgr is the singleton resource manager
var Mgr = NewManager()

// closable is implemented by resources that should be closed at peer shutdown.
type closable interface {
	Close()
}

// channelNotifier is implemented by resources that wish to be informed when a peer joins a channel
type channelNotifier interface {
	ChannelJoined(channelID string)
}

// Manager initializes resources at peer startup and closes them at peer shutdown.
type Manager struct {
	mutex      sync.RWMutex
	creators   []interface{}
	resources  []interface{}
	joinedChan chan string
}

// NewManager returns a new resource Manager
func NewManager() *Manager {
	m := &Manager{
		joinedChan: make(chan string, 1),
	}
	go m.listenChannel()
	return m
}

// Register is a convenience function that registers a resource creator function.
// See Manager.Register for details.
func Register(creator interface{}) {
	Mgr.Register(creator)
}

// Register registers a resource creator function. The function is invoked on peer startup.
// The creator function defines a set of arguments (which must all be of type interface{},
// struct{}, or *struct{}) and returns a single pointer to a struct. Each of the args are resolved
// against a set of registered providers. If a provider is found then it is passed in as the argument
// value. If a provider is not found then a panic results.
//
// Example:
//
//	func CreateMyResource(x someProvider, y someOtherProvider) *MyStruct {
//		return &MyResource{x: x, y: y}
//	}
//
//	Register(CreateMyResource)
//
func (b *Manager) Register(creator interface{}) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.creators = append(b.creators, creator)
}

// Initialize initializes all of the registered resources with the given set of providers.
func (b *Manager) Initialize(providers ...interface{}) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Add all of the providers since they are also considered to be resources
	b.resources = append(b.resources, providers...)

	invoker := injectinvoker.New(providers...)

	attempt := 0
	creators := b.creators
	for len(creators) > 0 {
		resources, unsatisfied, err := b.initialize(invoker, creators)
		attempt++
		if len(unsatisfied) == 0 {
			logger.Debugf("All resource providers were resolved on attempt #%d", attempt)
		} else {
			logger.Debugf("%d of %d resource providers were resolved on attempt #%d", len(resources), len(creators), attempt)
		}

		creators = unsatisfied

		b.resources = append(b.resources, resources...)

		if err != nil && (errors.Cause(err) != injectinvoker.ErrProviderNotFound || len(resources) == 0) {
			return err
		}
	}

	return nil
}

// Resources returns all resources
func (b *Manager) Resources() []interface{} {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.resources
}

func (b *Manager) initialize(invoker *injectinvoker.Invoker, creators []interface{}) ([]interface{}, []interface{}, error) {
	var resources []interface{}
	var unsatisfied []interface{}
	var errLast error

	for _, creator := range creators {
		r, err := create(invoker, creator)
		if err != nil {
			if errors.Cause(err) == injectinvoker.ErrProviderNotFound {
				unsatisfied = append(unsatisfied, creator)
				errLast = errors.WithMessagef(err, "Error creating resource [%s]", reflect.TypeOf(creator))
				continue
			}
			return nil, nil, errors.WithMessagef(err, "Error creating resource [%s]", reflect.TypeOf(creator))
		}
		resources = append(resources, r)

		// Add the resource to the set of providers to be available
		// for subsequent resources' dependencies
		invoker.AddProvider(r)
	}
	return resources, unsatisfied, errLast
}

// ChannelJoined notifies all applicable resources that the peer has joined a channel
func (b *Manager) ChannelJoined(channelID string) {
	b.joinedChan <- channelID
}

// Close closes all applicable resources (i.e. the ones that implement closable).
func (b *Manager) Close() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	close(b.joinedChan)
	for _, r := range b.resources {
		doClose(r)
	}
}

func (b *Manager) listenChannel() {
	for channelID := range b.joinedChan {
		b.notifyChannelJoined(channelID)
	}
}

func (b *Manager) notifyChannelJoined(channelID string) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for _, r := range b.resources {
		notifyChannelJoined(r, channelID)
	}
}

func create(invoker *injectinvoker.Invoker, fctn interface{}) (interface{}, error) {
	retVals, err := invoker.Invoke(fctn)
	if err != nil {
		return nil, err
	}

	// The function must return exactly one value, which is the Resource
	if len(retVals) != 1 {
		return nil, errors.New("invalid number of return values - expecting return value [Resource]")
	}

	return retVals[0].Interface(), nil
}

func doClose(res interface{}) {
	c, ok := res.(closable)
	if ok {
		logger.Debugf("Notifying resource %s that we're shutting down", reflect.TypeOf(res))
		c.Close()
	}
}

func notifyChannelJoined(res interface{}, channelID string) {
	n, ok := res.(channelNotifier)
	if ok {
		logger.Debugf("Notifying resource %s that channel [%s] was joined", reflect.TypeOf(res), channelID)
		n.ChannelJoined(channelID)
	}
}
