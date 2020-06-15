/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scc

import (
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/scc"
	"github.com/pkg/errors"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/injectinvoker"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

var logger = flogging.MustGetLogger("ext_scc")

var instance = newRegistry()

// Create returns a list of system chain codes, initialized with the given providers.
func Create(providers ...interface{}) []scc.SelfDescribingSysCC {
	instance.registerChaincodes(providers)

	return instance.chaincodes()
}

// Registry maintains a registry of in-process system chaincode
type Registry struct {
	mutex    sync.RWMutex
	creators []interface{}
	registry map[string]scc.SelfDescribingSysCC
}

func newRegistry() *Registry {
	r := &Registry{
		registry: make(map[string]scc.SelfDescribingSysCC),
	}
	resource.Register(r.Initialize)
	return r
}

// Register registers a System Chaincode creator function. The system chaincode
// will be initialized during peer startup with all of its declared dependencies.
func Register(ccCreator interface{}) {
	instance.addCreator(ccCreator)
}

// Initialize is called on peer startup
func (r *Registry) Initialize() *Registry {
	logger.Info("Initializing in-process system chaincode registry")

	return r
}

type channelListener interface {
	ChannelJoined(channelID string)
}

// ChannelJoined is called when the peer joins a channel
func (r *Registry) ChannelJoined(channelID string) {
	logger.Infof("[%s] Channel joined", channelID)

	for _, cc := range r.chaincodes() {
		l, ok := cc.(channelListener)
		if ok {
			logger.Infof(" [%s] Notifying in-process system chaincode [%s] that channel was joined", channelID, cc.Name())

			l.ChannelJoined(channelID)
		}
	}
}

func (r *Registry) registerChaincodes(providers []interface{}) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	b := injectinvoker.NewBuilder()
	for _, c := range r.creators {
		b.Add(c)
	}

	descs, err := b.Build(append(providers, resource.Mgr.Resources()...)...)
	if err != nil {
		panic(err)
	}

	logger.Infof("Registering [%d] in-process system chaincodes", len(descs))

	for _, desc := range descs {
		err = r.register(desc.(scc.SelfDescribingSysCC))
		if err != nil {
			panic(err)
		}
	}
}

func (r *Registry) addCreator(c interface{}) {
	r.creators = append(r.creators, c)
}

func (r *Registry) register(cc scc.SelfDescribingSysCC) error {
	logger.Infof("Registering in-process system chaincode [%s]", cc.Name())

	_, exists := r.registry[cc.Name()]
	if exists {
		return errors.Errorf("system chaincode already registered: [%s]", cc.Name())
	}

	r.registry[cc.Name()] = cc

	return nil
}

func (r *Registry) chaincodes() []scc.SelfDescribingSysCC {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var ccs []scc.SelfDescribingSysCC

	for _, cc := range r.registry {
		ccs = append(ccs, cc)
	}

	return ccs
}
