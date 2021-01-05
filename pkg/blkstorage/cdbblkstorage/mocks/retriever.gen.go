// Code generated by counterfeiter. DO NOT EDIT.
package mocks

import (
	"sync"

	"github.com/hyperledger/fabric-protos-go/common"
)

type Retriever struct {
	RetrieveBlockByNumberStub        func(blockNum uint64) (*common.Block, error)
	retrieveBlockByNumberMutex       sync.RWMutex
	retrieveBlockByNumberArgsForCall []struct {
		blockNum uint64
	}
	retrieveBlockByNumberReturns struct {
		result1 *common.Block
		result2 error
	}
	retrieveBlockByNumberReturnsOnCall map[int]struct {
		result1 *common.Block
		result2 error
	}
	RetrieveBlockByHashStub        func(blockHash []byte) (*common.Block, error)
	retrieveBlockByHashMutex       sync.RWMutex
	retrieveBlockByHashArgsForCall []struct {
		blockHash []byte
	}
	retrieveBlockByHashReturns struct {
		result1 *common.Block
		result2 error
	}
	retrieveBlockByHashReturnsOnCall map[int]struct {
		result1 *common.Block
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *Retriever) RetrieveBlockByNumber(blockNum uint64) (*common.Block, error) {
	fake.retrieveBlockByNumberMutex.Lock()
	ret, specificReturn := fake.retrieveBlockByNumberReturnsOnCall[len(fake.retrieveBlockByNumberArgsForCall)]
	fake.retrieveBlockByNumberArgsForCall = append(fake.retrieveBlockByNumberArgsForCall, struct {
		blockNum uint64
	}{blockNum})
	fake.recordInvocation("RetrieveBlockByNumber", []interface{}{blockNum})
	fake.retrieveBlockByNumberMutex.Unlock()
	if fake.RetrieveBlockByNumberStub != nil {
		return fake.RetrieveBlockByNumberStub(blockNum)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.retrieveBlockByNumberReturns.result1, fake.retrieveBlockByNumberReturns.result2
}

func (fake *Retriever) RetrieveBlockByNumberCallCount() int {
	fake.retrieveBlockByNumberMutex.RLock()
	defer fake.retrieveBlockByNumberMutex.RUnlock()
	return len(fake.retrieveBlockByNumberArgsForCall)
}

func (fake *Retriever) RetrieveBlockByNumberArgsForCall(i int) uint64 {
	fake.retrieveBlockByNumberMutex.RLock()
	defer fake.retrieveBlockByNumberMutex.RUnlock()
	return fake.retrieveBlockByNumberArgsForCall[i].blockNum
}

func (fake *Retriever) RetrieveBlockByNumberReturns(result1 *common.Block, result2 error) {
	fake.RetrieveBlockByNumberStub = nil
	fake.retrieveBlockByNumberReturns = struct {
		result1 *common.Block
		result2 error
	}{result1, result2}
}

func (fake *Retriever) RetrieveBlockByNumberReturnsOnCall(i int, result1 *common.Block, result2 error) {
	fake.RetrieveBlockByNumberStub = nil
	if fake.retrieveBlockByNumberReturnsOnCall == nil {
		fake.retrieveBlockByNumberReturnsOnCall = make(map[int]struct {
			result1 *common.Block
			result2 error
		})
	}
	fake.retrieveBlockByNumberReturnsOnCall[i] = struct {
		result1 *common.Block
		result2 error
	}{result1, result2}
}

func (fake *Retriever) RetrieveBlockByHash(blockHash []byte) (*common.Block, error) {
	var blockHashCopy []byte
	if blockHash != nil {
		blockHashCopy = make([]byte, len(blockHash))
		copy(blockHashCopy, blockHash)
	}
	fake.retrieveBlockByHashMutex.Lock()
	ret, specificReturn := fake.retrieveBlockByHashReturnsOnCall[len(fake.retrieveBlockByHashArgsForCall)]
	fake.retrieveBlockByHashArgsForCall = append(fake.retrieveBlockByHashArgsForCall, struct {
		blockHash []byte
	}{blockHashCopy})
	fake.recordInvocation("RetrieveBlockByHash", []interface{}{blockHashCopy})
	fake.retrieveBlockByHashMutex.Unlock()
	if fake.RetrieveBlockByHashStub != nil {
		return fake.RetrieveBlockByHashStub(blockHash)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.retrieveBlockByHashReturns.result1, fake.retrieveBlockByHashReturns.result2
}

func (fake *Retriever) RetrieveBlockByHashCallCount() int {
	fake.retrieveBlockByHashMutex.RLock()
	defer fake.retrieveBlockByHashMutex.RUnlock()
	return len(fake.retrieveBlockByHashArgsForCall)
}

func (fake *Retriever) RetrieveBlockByHashArgsForCall(i int) []byte {
	fake.retrieveBlockByHashMutex.RLock()
	defer fake.retrieveBlockByHashMutex.RUnlock()
	return fake.retrieveBlockByHashArgsForCall[i].blockHash
}

func (fake *Retriever) RetrieveBlockByHashReturns(result1 *common.Block, result2 error) {
	fake.RetrieveBlockByHashStub = nil
	fake.retrieveBlockByHashReturns = struct {
		result1 *common.Block
		result2 error
	}{result1, result2}
}

func (fake *Retriever) RetrieveBlockByHashReturnsOnCall(i int, result1 *common.Block, result2 error) {
	fake.RetrieveBlockByHashStub = nil
	if fake.retrieveBlockByHashReturnsOnCall == nil {
		fake.retrieveBlockByHashReturnsOnCall = make(map[int]struct {
			result1 *common.Block
			result2 error
		})
	}
	fake.retrieveBlockByHashReturnsOnCall[i] = struct {
		result1 *common.Block
		result2 error
	}{result1, result2}
}

func (fake *Retriever) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.retrieveBlockByNumberMutex.RLock()
	defer fake.retrieveBlockByNumberMutex.RUnlock()
	fake.retrieveBlockByHashMutex.RLock()
	defer fake.retrieveBlockByHashMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *Retriever) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}