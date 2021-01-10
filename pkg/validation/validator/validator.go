/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"context"
	"fmt"
	"sync"
	"time"

	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/committer/txvalidator/plugin"
	validatorv20 "github.com/hyperledger/fabric/core/committer/txvalidator/v20"
	"github.com/hyperledger/fabric/core/committer/txvalidator/v20/plugindispatcher"
	"github.com/hyperledger/fabric/core/ledger"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/txflags"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	vcommon "github.com/trustbloc/fabric-peer-ext/pkg/validation/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validationpolicy"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validationresults"
)

var logger = flogging.MustGetLogger("ext_validation")

// traceLogger is used for very detailed logging
var traceLogger = flogging.MustGetLogger("ext_validation_trace")

// ignoreCancel is a cancel function that does nothing
var ignoreCancel = func() {}

var instance *Provider
var mutex sync.RWMutex

// NewTxValidator creates a new validator for the given channel
func NewTxValidator(
	channelID string,
	sem semaphore,
	cr channelResources,
	ler ledgerResources,
	lcr plugindispatcher.LifecycleResources,
	cor plugindispatcher.CollectionResources,
	pm plugin.Mapper,
	cpmg policies.ChannelPolicyManagerGetter,
	cp bccsp.BCCSP) txvalidator.Validator {
	mutex.RLock()
	defer mutex.RUnlock()

	return instance.createValidator(channelID, sem, cr, ler, lcr, cor, pm, cpmg, cp)
}

type semaphore interface {
	// Acquire implements semaphore-like acquire semantics
	Acquire(ctx context.Context) error

	// Release implements semaphore-like release semantics
	Release()
}

type channelResources interface {
	MSPManager() msp.MSPManager
	Apply(configtx *cb.ConfigEnvelope) error
	GetMSPIDs() []string
	Capabilities() channelconfig.ApplicationCapabilities
}

type ledgerResources interface {
	GetTransactionByID(txID string) (*pb.ProcessedTransaction, error)
	NewQueryExecutor() (ledger.QueryExecutor, error)
}

type identityDeserializerProvider interface {
	GetIdentityDeserializer(channelID string) msp.IdentityDeserializer
}

type txValidator interface {
	ValidateTx(req *validatorv20.BlockValidationRequest, results chan<- *validatorv20.BlockValidationResult)
}

type validator struct {
	txValidator
	*discovery.Discovery
	channelID             string
	resultsCache          *validationresults.Cache
	resultsChan           chan *validationresults.Results
	validationPolicy      *validationpolicy.PolicyEvaluator
	validationMinWaitTime time.Duration
	semaphore             semaphore
}

// Providers contains the dependencies for the validator
type Providers struct {
	Gossip gossipProvider
	Idp    identityDeserializerProvider
}

// Provider maintains a set of V2 transaction validators, one per channel
type Provider struct {
	*Providers
	mutex      sync.RWMutex
	validators map[string]*validator
}

type gossipProvider interface {
	GetGossipService() gossipapi.GossipService
}

// NewProvider returns a new validator provider
func NewProvider(providers *Providers) *Provider {
	logger.Info("Creating validator provider")

	mutex.Lock()
	defer mutex.Unlock()

	instance = &Provider{
		Providers:  providers,
		validators: make(map[string]*validator),
	}

	return instance
}

func (p *Provider) createValidator(
	channelID string,
	sem semaphore,
	cr channelResources,
	ler ledgerResources,
	lcr plugindispatcher.LifecycleResources,
	cor plugindispatcher.CollectionResources,
	pm plugin.Mapper,
	cpmg policies.ChannelPolicyManagerGetter,
	cp bccsp.BCCSP) *validator {

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, exists := p.validators[channelID]; exists {
		// Should never happen
		panic(fmt.Errorf("a validator already exists for channel [%s]", channelID))
	}

	disc := discovery.New(channelID, p.Gossip.GetGossipService())

	v := &validator{
		channelID:             channelID,
		Discovery:             disc,
		resultsChan:           make(chan *validationresults.Results, 10),
		resultsCache:          validationresults.NewCache(),
		txValidator:           validatorv20.NewTxValidator(channelID, sem, cr, ler, lcr, cor, pm, cpmg, cp),
		validationPolicy:      validationpolicy.New(channelID, disc, cauthdsl.NewPolicyProvider(p.Idp.GetIdentityDeserializer(channelID))),
		validationMinWaitTime: config.GetValidationWaitTime(),
		semaphore:             sem,
	}

	p.validators[channelID] = v

	return v
}

// GetValidatorForChannel returns the validator for the given channel
func (p *Provider) GetValidatorForChannel(channelID string) vcommon.DistributedValidator {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.validators[channelID]
}

// GetValidatingPeers returns the peers that are involved in validating the given block
func (v *validator) GetValidatingPeers(block *cb.Block) (discovery.PeerGroup, error) {
	return v.validationPolicy.GetValidatingPeers(block)
}

// Validate performs validation of the given block. The block is updated with the validation results.
func (v *validator) Validate(block *cb.Block) error {
	startValidation := time.Now() // timer to log ValidateResults block duration
	logger.Debugf("[%s] Starting validation for block [%d]", v.channelID, block.Header.Number)

	// Initialize the txResults all to TxValidationCode_NOT_VALIDATED
	protoutil.InitBlockMetadata(block)
	block.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER] = txflags.New(len(block.Data.Data))

	validatingPeers, err := v.GetValidatingPeers(block)
	if err != nil {
		logger.Warningf("[%s] Error getting validating peers for block %d: %s", v.channelID, block.Header.Number, err)
	} else if validatingPeers.ContainsLocal() {
		go v.validateLocal(block)
	}

	txResults := newTxResults(v.channelID, block)

	err = v.waitForValidationResults(ignoreCancel, block.Header.Number, txResults, v.validationMinWaitTime)
	if err != nil {
		// Log a warning and continue validating the remaining transactions ourselves
		logger.Warningf("[%s] Got error in validation response for block %d: %s", v.channelID, block.Header.Number, err)
	}

	notValidated := txResults.UnvalidatedMap()
	if len(notValidated) > 0 {
		ctx, cancel := context.WithCancel(context.Background())

		// Haven't received results for some of the transactions. ValidateResults the remaining ones.
		go v.validateRemaining(ctx, block, notValidated)

		// Wait forever for a response
		err := v.waitForValidationResults(cancel, block.Header.Number, txResults, time.Hour)
		if err != nil {
			logger.Warningf("[%s] Got error validating remaining transactions in block %d: %s", v.channelID, block.Header.Number, err)
			return err
		}
	}

	// make sure no transaction has skipped validation
	if !txResults.AllValidated() {
		logger.Errorf("[%s] Not all transactions in block %d were validated", v.channelID, block.Header.Number)

		return errors.Errorf("Not all transactions in block %d were validated", block.Header.Number)
	}

	// Set the transaction status codes in the block's metadata
	protoutil.InitBlockMetadata(block)

	block.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER] = txResults.Flags()

	logger.Infof("[%s] Validated block [%d] in %dms", v.channelID, block.Header.Number, time.Since(startValidation).Milliseconds())

	return nil
}

// ValidatePartial partially validates the block and sends the validation results over Gossip.
// Note that this function is only called by validators and not committers.
func (v *validator) ValidatePartial(ctx context.Context, block *cb.Block) (txflags.ValidationFlags, []string, error) {
	// Initialize the flags all to TxValidationCode_NOT_VALIDATED
	protoutil.InitBlockMetadata(block)
	block.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER] = txflags.New(len(block.Data.Data))

	numValidated, txFlags, txIDs, err := v.validateBlock(ctx, block, v.validationPolicy.GetTxFilter(block))
	if err != nil {
		if err == context.Canceled {
			logger.Debugf("[%s] ... validation of block %d was cancelled", v.channelID, block.Header.Number)

			return nil, nil, err
		}

		logger.Infof("[%s] Got error in validation of block %d: %s", v.channelID, block.Header.Number, err)

		return nil, nil, err
	}

	logger.Infof("[%s] ... finished validating %d transactions in block %d. Error: %v", v.channelID, numValidated, block.Header.Number, err)

	return txFlags, txIDs, nil
}

// SubmitValidationResults is called by the Gossip handler when it receives validation results from a remote peer.
// The committer merges these results with results from other peers.
func (v *validator) SubmitValidationResults(results *validationresults.Results) {
	logger.Debugf("[%s] Got validation results from %s for block %d", v.channelID, results.Endpoint, results.BlockNumber)

	v.resultsChan <- results
}

// validateBlock validates the transactions in the block for which this peer is responsible (according to validation policy) and
// returns the results.
func (v *validator) validateBlock(ctx context.Context, block *cb.Block, shouldValidate validationpolicy.TxFilter) (int, txflags.ValidationFlags, []string, error) {
	results := make(chan *validatorv20.BlockValidationResult)

	transactions := v.getTransactionsToValidate(block, shouldValidate)

	// validateBlock transactions in the background. The results are posted to the given results channel
	go v.validateTransactions(ctx, block, transactions, results)

	logger.Debugf("[%s] Expecting %d validation responses for block %d", v.channelID, len(transactions), block.Header.Number)

	var err error
	var errPos int

	// Initialize transaction as valid here, then set invalidation reason code upon invalidation below
	txsfltr := txflags.New(len(block.Data.Data))
	txidArray := make([]string, len(block.Data.Data))

	// now we read responses in the order in which they come back
	for i := 0; i < len(transactions); i++ {
		res := <-results

		if res.Err != nil {
			// if there is an error, we buffer its Flags, wait for
			// all workers to complete validation and then return
			// the error from the first tx in this block that returned an error

			if err == nil || res.TIdx < errPos {
				err = res.Err
				errPos = res.TIdx

				if err == context.Canceled {
					logger.Debugf("[%s] Validation of block %d was cancelled", v.channelID, block.Header.Number)
				} else {
					logger.Warningf("[%s] Got error for idx %d in block %d: %s", v.channelID, res.TIdx, block.Header.Number, err)
				}
			}
		} else {
			traceLogger.Debugf("[%s] Got result for idx %d in block %d: %s", v.channelID, res.TIdx, block.Header.Number, res.ValidationCode)

			txsfltr.SetFlag(res.TIdx, res.ValidationCode)

			if res.ValidationCode == pb.TxValidationCode_VALID {
				txidArray[res.TIdx] = res.Txid
			}
		}
	}

	return len(transactions), txsfltr, txidArray, err
}

func (v *validator) validateTransactions(ctx context.Context, block *cb.Block, transactions map[int]struct{}, results chan *validatorv20.BlockValidationResult) {
	var err error

	for txIdx, d := range block.Data.Data {
		_, ok := transactions[txIdx]
		if !ok {
			continue
		}

		// Only attempt to acquire the semaphore if there was no previous error
		if err == nil {
			// ensure that we don't have too many concurrent validation workers
			err = v.semaphore.Acquire(ctx)
		}

		if err == nil {
			go func(index int, data []byte) {
				defer v.semaphore.Release()

				v.ValidateTx(&validatorv20.BlockValidationRequest{
					D:     data,
					Block: block,
					TIdx:  index,
				}, results)
			}(txIdx, d)
		} else {
			// Send an error response for the transaction index
			results <- &validatorv20.BlockValidationResult{
				TIdx: txIdx,
				Err:  err,
			}
		}
	}
}

// validateLocal is called by the committer to validateBlock a portion of the block and submits the results to the results channel.
// Note that this function is only called if the committer is also a validator.
func (v *validator) validateLocal(block *cb.Block) {
	logger.Debugf("[%s] This committer is also a validator. Starting validation of transactions in block %d", v.channelID, block.Header.Number)

	var errStr string

	numValidated, txFlags, txIDs, err := v.validateBlock(context.Background(), block, v.validationPolicy.GetTxFilter(block))
	if err != nil {
		if err == context.Canceled {
			logger.Debugf("[%s] ... validation of block %d was cancelled", v.channelID, block.Header.Number)

			return
		}

		errStr = err.Error()

		logger.Infof("[%s] Got error validating transactions in block %d: %s", v.channelID, block.Header.Number, err)
	}

	logger.Infof("[%s] ... finished validating %d transactions in block %d", v.channelID, numValidated, block.Header.Number)

	v.resultsChan <- &validationresults.Results{
		BlockNumber: block.Header.Number,
		TxFlags:     txFlags,
		TxIDs:       txIDs,
		Err:         errStr,
		Local:       true,
		Endpoint:    v.Self().Endpoint,
		MSPID:       v.Self().MSPID,
	}
}

// validateRemaining is called by the committer to validateBlock any transactions in the block that have not yet been validated
// and submits the results to the results channel.
func (v *validator) validateRemaining(ctx context.Context, block *cb.Block, notValidated map[int]struct{}) {
	logger.Debugf("[%s] Starting validation of %d of %d transactions in block %d that were not validated ...", v.channelID, len(notValidated), len(block.Data.Data), block.Header.Number)

	numValidated, txFlags, txIDs, err := v.validateBlock(ctx, block,
		func(txIdx int) bool {
			_, ok := notValidated[txIdx]
			return ok
		},
	)

	var errStr string
	if err != nil {
		if err == context.Canceled {
			logger.Debugf("[%s] ... validation of %d of %d transactions in block %d that were not validated was cancelled", v.channelID, numValidated, len(block.Data.Data), block.Header.Number)

			return
		}

		errStr = err.Error()
	}

	logger.Infof("[%s] ... finished validating %d of %d transactions in block %d that were not validated. Err: %v", v.channelID, numValidated, len(block.Data.Data), block.Header.Number, err)

	self := v.Self()

	v.resultsChan <- &validationresults.Results{
		BlockNumber: block.Header.Number,
		TxFlags:     txFlags,
		TxIDs:       txIDs,
		Err:         errStr,
		Local:       true,
		Endpoint:    self.Endpoint,
		MSPID:       self.MSPID,
	}
}

// waitForValidationResults is called by the committer to accumulate validation results from various peers. This function
// returns after all transactions in the block are validated or if a timeout occurs.
func (v *validator) waitForValidationResults(cancel context.CancelFunc, blockNumber uint64, txResults *txResults, timeout time.Duration) error {
	logger.Infof("[%s] Waiting up to %s for validation responses for block %d ...", v.channelID, timeout, blockNumber)

	start := time.Now()
	timeoutChan := time.After(timeout)

	// See if there are cached results for the current block
	results := v.resultsCache.Remove(blockNumber)
	if len(results) > 0 {
		go func() {
			for _, result := range results {
				logger.Debugf("[%s] Retrieved validation results from the cache: %s", v.channelID, result)

				v.resultsChan <- result
			}
		}()
	}

	for {
		select {
		case result := <-v.resultsChan:
			logger.Infof("[%s] Got results from [%s] for block %d after %s", v.channelID, result.Endpoint, result.BlockNumber, time.Since(start))

			done, err := v.handleResults(blockNumber, txResults, result)
			if err != nil {
				logger.Infof("[%s] Received error in validation results from [%s] peer for block %d: %s", v.channelID, result.Endpoint, result.BlockNumber, err)

				return err
			}

			if done {
				// Cancel any background validations
				cancel()

				logger.Debugf("[%s] Block %d is all validated. Done waiting %s for responses.", v.channelID, blockNumber, time.Since(start))

				return nil
			}
		case <-timeoutChan:
			logger.Debugf("[%s] Timed out after %s waiting for validation response for block %d", v.channelID, timeout, blockNumber)

			return nil
		}
	}
}

func (v *validator) handleResults(blockNumber uint64, txResults *txResults, result *validationresults.Results) (done bool, err error) {
	if result.BlockNumber < blockNumber {
		logger.Debugf("[%s] Discarding validation results from [%s] for block %d since we're waiting on block %d", v.channelID, result.Endpoint, result.BlockNumber, blockNumber)

		return false, nil
	}

	if result.BlockNumber > blockNumber {
		// We got results for a block that's greater than the current block. Cache the results and use them when we get to that block.
		logger.Debugf("[%s] Got validation results from [%s] for block %d but we're waiting on block %d. Caching the results.", v.channelID, result.Endpoint, result.BlockNumber, blockNumber)

		v.resultsCache.Add(result)

		return false, nil
	}

	if result.Err != "" {
		if result.Err == context.Canceled.Error() {
			// Ignore this error
			logger.Debugf("[%s] Validation was canceled in [%s] peer for block %d", v.channelID, result.Endpoint, result.BlockNumber)

			return false, nil
		}

		return false, fmt.Errorf(result.Err)
	}

	results := v.resultsCache.Add(result)
	err = v.validationPolicy.ValidateResults(results)
	if err != nil {
		logger.Debugf("[%s] Validation policy NOT satisfied for block %d, Results: %s, Error: %s", v.channelID, result.BlockNumber, results, err)

		return false, nil
	}

	done, err = txResults.Merge(result.TxFlags, result.TxIDs)
	if err != nil {
		return false, err
	}

	logger.Debugf("[%s] Validation policy satisfied for block %d, Results: %s, Done: %t", v.channelID, result.BlockNumber, results, done)

	return done, nil
}

func (v *validator) getTransactionsToValidate(block *cb.Block, shouldValidate validationpolicy.TxFilter) map[int]struct{} {
	transactions := make(map[int]struct{})

	blockFltr := txflags.ValidationFlags(block.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER])

	for txIdx := range block.Data.Data {
		if !shouldValidate(txIdx) {
			continue
		}

		txStatus := blockFltr.Flag(txIdx)
		if txStatus != pb.TxValidationCode_NOT_VALIDATED {
			traceLogger.Debugf("[%s] Not validating TxIdx[%d] in block %d since it has already been set to %s", v.channelID, txIdx, block.Header.Number, txStatus)

			continue
		}

		transactions[txIdx] = struct{}{}
	}

	return transactions
}
