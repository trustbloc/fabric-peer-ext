/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resource

import (
	"reflect"
	"sort"
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
	mutex         sync.RWMutex
	registrations registrations
	resources     []interface{}
	joinedChan    chan string
	order         int
}

// NewManager returns a new resource Manager
func NewManager() *Manager {
	m := &Manager{
		joinedChan: make(chan string, 1),
	}
	go m.listenChannel()
	return m
}

// Priority is the relative priority of the resource
type Priority uint

const (
	// PriorityHighest indicates that the resource should be initialized first
	PriorityHighest Priority = 0

	// PriorityHigh indicates that the resource should be among the first to be initialized
	PriorityHigh Priority = 10

	// PriorityAboveNormal indicates that the resource should be initialized before normal resources
	PriorityAboveNormal Priority = 25

	// PriorityNormal indicates that the resource can be initialized in any order
	PriorityNormal Priority = 50

	// PriorityBelowNormal indicates that the resource should be initialized after normal resources
	PriorityBelowNormal Priority = 75

	// PriorityLow indicates that the resource should be among the last to be initialized
	PriorityLow Priority = 90

	// PriorityLowest indicates that the resource should be initialized last
	PriorityLowest Priority = 100
)

// Register is a convenience function that registers a resource creator function.
// See Manager.Register for details.
func Register(creator interface{}, priority Priority) {
	Mgr.Register(creator, priority)
}

// Register registers a resource creator function. The function is invoked on peer startup.
// The creator function defines a set of arguments (which must all be of type interface{},
// struct{}, or *struct{}) and returns a single pointer to a struct. Each of the args are resolved
// against a set of registered providers. If a provider is found then it is passed in as the argument
// value. If a provider is not found then a panic results.
//
// Argument, priority, specifies the order in which the resource should be initialized
// relative to other resources.
//
// Example:
//
//	func CreateMyResource(x someProvider, y someOtherProvider) *MyStruct {
//		return &MyResource{x: x, y: y}
//	}
//
//	Register(CreateMyResource, resource.PriorityNormal)
//
func (b *Manager) Register(creator interface{}, priority Priority) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.order++
	b.registrations = append(b.registrations, newRegistration(creator, priority, b.order))
}

// Initialize initializes all of the registered resources with the given set of providers.
func (b *Manager) Initialize(providers ...interface{}) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	invoker := injectinvoker.New(providers...)
	for _, creator := range b.registrations.sort().creators() {
		r, err := create(invoker, creator)
		if err != nil {
			return errors.WithMessagef(err, "Error creating resource [%s]", reflect.TypeOf(creator))
		}
		b.resources = append(b.resources, r)

		// Add the resource to the set of providers to be available
		// for subsequent resources' dependencies
		invoker.AddProvider(r)
	}
	return nil
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

type registration struct {
	creator  interface{}
	priority Priority
	order    int
}

func newRegistration(creator interface{}, priority Priority, order int) *registration {
	return &registration{creator: creator, priority: priority, order: order}
}

type registrations []*registration

// sort sorts the resources first by priority and then by the
// order in which the resources were registered.
func (r registrations) sort() registrations {
	sort.Slice(r, func(i, j int) bool {
		ri := r[i]
		rj := r[j]
		if ri.priority == rj.priority {
			return ri.order < rj.order
		}
		return ri.priority < rj.priority
	})
	return r
}

func (r registrations) creators() []interface{} {
	creators := make([]interface{}, len(r))
	for i, entry := range r {
		creators[i] = entry.creator
	}
	return creators
}
