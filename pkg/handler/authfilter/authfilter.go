/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package authfilter

import (
	"context"
	"sync"

	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/handlers/auth"
	"github.com/pkg/errors"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/injectinvoker"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

var logger = flogging.MustGetLogger("ext_authfilter")

var instance = newRegistry()

// Register registers an Auth filter under the given name
func Register(creator interface{}) {
	instance.addCreator(creator)
}

// Get returns an Auth filter if registered under the given name; nil otherwise
func Get(name string) auth.Filter {
	return instance.get(name)
}

type registry struct {
	authFilters map[string]auth.Filter
	mutex       sync.RWMutex
	creators    []interface{}
}

func newRegistry() *registry {
	r := &registry{
		authFilters: make(map[string]auth.Filter),
	}

	resource.Register(r.Initialize)

	return r
}

// Initialize is called on peer startup
func (r *registry) Initialize() *registry {
	logger.Info("Initializing auth filter registry ...")

	// Acquire a write lock. The lock will be released once
	//the auth filters have all registered.
	r.mutex.Lock()

	go r.registerAuthFilters()

	return r
}

func (r *registry) addCreator(c interface{}) {
	r.creators = append(r.creators, c)
}

type handler interface {
	Name() string
	Init(peer.EndorserServer)
	ProcessProposal(context.Context, *peer.SignedProposal) (*peer.ProposalResponse, error)
}

func (r *registry) registerAuthFilters() {
	// mutex.Lock was called by the Initialize function and this function
	// will call mutex.Unlock after all auth filters are registered.
	defer r.mutex.Unlock()

	b := injectinvoker.NewBuilder()
	for _, c := range r.creators {
		b.Add(c)
	}

	filters, err := b.Build(resource.Mgr.Resources()...)
	if err != nil {
		panic(err)
	}

	logger.Infof("... registering %d auth filter(s)", len(filters))

	for _, f := range filters {
		r.register(f.(handler))
	}

	logger.Info("... finished registering auth filters")
}

func (r *registry) register(h handler) {
	_, ok := r.authFilters[h.Name()]
	if ok {
		panic(errors.Errorf("auth filter [%s] already registered", h.Name()))
	}

	logger.Infof("Registering auth filter [%s]", h.Name())

	r.authFilters[h.Name()] = h
}

func (r *registry) get(name string) auth.Filter {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	f, ok := r.authFilters[name]
	if !ok {
		logger.Infof("Auth filter not registered: [%s]", name)
		return nil
	}

	logger.Infof("Returning auth filter: [%s]", name)

	return f
}
