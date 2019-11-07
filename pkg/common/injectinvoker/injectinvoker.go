/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package injectinvoker

import (
	"reflect"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("injectinvoker")

// ErrUnsupportedType indicates that the type of arg/field is not a struct or an interface
var ErrUnsupportedType = errors.New("unsupported type")

// ErrProviderNotFound indicates that a provider was not found for the interface
var ErrProviderNotFound = errors.New("no provider found that satisfies interface")

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
		p, err := inv.getProviderForType(argType)
		if err != nil {
			return nil, err
		}
		args[i] = p
	}
	return args, nil
}

func (inv *Invoker) getProviderForType(t reflect.Type) (reflect.Value, error) {
	switch t.Kind() {
	case reflect.Interface:
		return inv.getProviderForInterface(t)
	case reflect.Struct:
		return inv.getProviderForStruct(t)
	case reflect.Ptr:
		return inv.getProviderForStructPtr(t)
	default:
		return reflect.Value{}, errors.WithMessagef(ErrUnsupportedType, "argument [%s] must be either an interface or struct", t)
	}
}

func (inv *Invoker) getProviderForInterface(t reflect.Type) (reflect.Value, error) {
	for _, d := range inv.providers {
		pType := reflect.TypeOf(d)
		logger.Debugf("Checking if provider [%s] implements interface [%s] ...", pType, t)
		if pType.Implements(t) {
			logger.Debugf("... found provider [%s] which implements interface [%s]", pType, t)
			return reflect.ValueOf(d), nil
		}
		logger.Debugf("... provider [%s] does NOT implement interface [%s]", pType, t)
	}
	return reflect.Value{}, errors.WithMessagef(ErrProviderNotFound, "[%s]", t.String())
}

func (inv *Invoker) getProviderForStruct(t reflect.Type) (reflect.Value, error) {
	// Create an instance of the struct
	inst := reflect.New(t).Elem()
	err := inv.populateFields(t, inst)
	if err != nil {
		return reflect.Value{}, err
	}
	return inst, nil
}

func (inv *Invoker) getProviderForStructPtr(tPtr reflect.Type) (reflect.Value, error) {
	// Create an instance of the struct
	instPtr := reflect.New(tPtr.Elem())
	err := inv.populateFields(tPtr.Elem(), instPtr.Elem())
	if err != nil {
		return reflect.Value{}, err
	}
	return instPtr, nil
}

func (inv *Invoker) populateFields(t reflect.Type, inst reflect.Value) error {
	logger.Debugf("Checking fields of struct %s...", t)
	for i := 0; i < t.NumField(); i++ {
		err := inv.setProviderForField(t, inst, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (inv *Invoker) setProviderForField(t reflect.Type, inst reflect.Value, i int) error {
	field := t.Field(i)
	logger.Debugf("... checking field [%s] of type [%s]", field.Name, field.Type)
	instField := inst.Field(i)
	if !instField.CanSet() {
		logger.Debugf("... skipping field [%s] of type [%s] since the field cannot be set", field.Name, field.Type)
		return nil
	}

	p, err := inv.getProviderForType(field.Type)
	if err != nil {
		if errors.Cause(err) == ErrUnsupportedType {
			logger.Debugf("... skipping field [%s] of type [%s]", field.Name, field.Type)
			return nil
		}
		return err
	}

	logger.Debugf("... setting value of field [%s] of type [%s] on instance [%+v]", field.Name, field.Type, inst)
	instField.Set(p)

	return nil
}

func getArgTypes(fctn interface{}) []reflect.Type {
	fctnType := reflect.TypeOf(fctn)
	args := make([]reflect.Type, fctnType.NumIn())
	for i := 0; i < fctnType.NumIn(); i++ {
		args[i] = fctnType.In(i)
	}
	return args
}
