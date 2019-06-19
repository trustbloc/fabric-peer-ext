/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package injectinvoker

import (
	"fmt"
	"reflect"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("injectinvoker")

// Invoker is a dependency injecting function invoker that calls a given function,
// providing it with all required args.
type Invoker struct {
	providers []interface{}
}

// New returns a dependency injecting function invoker with the given set of providers.
func New(providers ...interface{}) *Invoker {
	return &Invoker{
		providers: providers,
	}
}

// AddProvider adds a provider to the set of providers
func (inv *Invoker) AddProvider(provider interface{}) {
	inv.providers = append(inv.providers, provider)
}

// Invoke calls the given function, providing it with the required providers.
// Each of the interface arguments specified in the given function is injected with a provider
// that implements that interface. An error is returned if no suitable provider is found for
// any of the supplied arguments.
// Note: All of the arguments in the provided function must be interfaces.
func (inv *Invoker) Invoke(fctn interface{}) ([]reflect.Value, error) {
	// Get the args required by the function
	args, err := inv.getArgValues(fctn)
	if err != nil {
		return nil, err
	}

	// Invoke the function with the provider args
	return reflect.ValueOf(fctn).Call(args), nil
}

func (inv *Invoker) getArgValues(fctn interface{}) ([]reflect.Value, error) {
	argTypes := getArgTypes(fctn)
	args := make([]reflect.Value, len(argTypes))
	for i, argType := range argTypes {
		if argType.Kind() != reflect.Interface {
			return nil, errors.Errorf("argument [%s] is not an interface", argType)
		}
		p, err := inv.getProviderForType(argType)
		if err != nil {
			return nil, err
		}
		args[i] = p
	}
	return args, nil
}

func (inv *Invoker) getProviderForType(t reflect.Type) (reflect.Value, error) {
	for _, d := range inv.providers {
		pType := reflect.TypeOf(d)
		logger.Infof("Checking if %s implements %s ...", pType, t)
		if pType.Implements(t) {
			logger.Infof("... %s does implement %s", pType, t)
			return reflect.ValueOf(d), nil
		}
		logger.Infof("... %s does NOT implement %s", pType, t)
	}
	return reflect.Value{}, fmt.Errorf("No provider found that satisfies interface [%s]", t.String())
}

func getArgTypes(fctn interface{}) []reflect.Type {
	fctnType := reflect.TypeOf(fctn)
	args := make([]reflect.Type, fctnType.NumIn())
	for i := 0; i < fctnType.NumIn(); i++ {
		args[i] = fctnType.In(i)
	}
	return args
}
