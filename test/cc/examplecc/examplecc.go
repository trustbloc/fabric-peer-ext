/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package examplecc

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	ccapi "github.com/hyperledger/fabric/extensions/chaincode/api"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"

	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas"
	dcasclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas/client"
)

type invokeFunc func(stub shim.ChaincodeStubInterface, args []string) pb.Response
type funcMap map[string]invokeFunc

const (
	v1 = "v1"

	warmupFunc             = "warmup"
	getFunc                = "get"
	putFunc                = "put"
	delFunc                = "del"
	putPrivateFunc         = "putprivate"
	getPrivateFunc         = "getprivate"
	queryPrivateFunc       = "queryprivate"
	putPrivateMultipleFunc = "putprivatemultiple"
	getPrivateMultipleFunc = "getprivatemultiple"
	delPrivateFunc         = "delprivate"
	getAndPutPrivateFunc   = "getandputprivate"
	putBothFunc            = "putboth"
	getAndPutBothFunc      = "getandputboth"
	invokeCCFunc           = "invokecc"
	getPrivateByRangeFunc  = "getprivatebyrange"
	putCASFunc             = "putcas"
	putCASMultipleFunc     = "putcasmultiple"
	getCASFunc             = "getcas"
	getAndPutCASFunc       = "getandputcas"
)

// DCASStubWrapperFactory creates a DCAS client wrapper around a stub
type DCASStubWrapperFactory interface {
	CreateDCASClientStubWrapper(coll string, stub shim.ChaincodeStubInterface) (dcasclient.DCAS, error)
}

// ExampleCC example chaincode that puts and gets state and private data
type ExampleCC struct {
	DCASStubWrapperFactory
	name         string
	dbArtifacts  map[string]*ccapi.DBArtifacts
	funcRegistry funcMap
}

// New returns a new example chaincode instance
func New(name string, dbArtifacts map[string]*ccapi.DBArtifacts, dcasClientFactory DCASStubWrapperFactory) *ExampleCC {
	cc := &ExampleCC{
		name:                   name,
		dbArtifacts:            dbArtifacts,
		DCASStubWrapperFactory: dcasClientFactory,
	}
	cc.initRegistry()
	return cc
}

// Name returns the name of this chaincode
func (cc *ExampleCC) Name() string { return cc.name }

// Version returns the version of the chaincode
func (cc *ExampleCC) Version() string { return v1 }

// Chaincode returns the DocumentCC chaincode
func (cc *ExampleCC) Chaincode() shim.Chaincode { return cc }

// GetDBArtifacts returns Couch DB indexes (if applicable)
func (cc *ExampleCC) GetDBArtifacts(collNames []string) map[string]*ccapi.DBArtifacts {
	dbArtifacts := make(map[string]*ccapi.DBArtifacts)

	for dbName, artifactsForDB := range cc.dbArtifacts {
		collIndexes := make(map[string][]string)
		for _, collName := range collNames {
			collIndexes[collName] = artifactsForDB.CollectionIndexes[collName]
		}

		dbArtifacts[dbName] = &ccapi.DBArtifacts{
			Indexes:           artifactsForDB.Indexes,
			CollectionIndexes: collIndexes,
		}
	}

	return dbArtifacts
}

// Init is not used
func (cc *ExampleCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke invoke the chaincode with a given function
func (cc *ExampleCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	if function == "" {
		return shim.Error("Expecting function")
	}

	f, ok := cc.funcRegistry[function]
	if !ok {
		return shim.Error(fmt.Sprintf("Unknown function [%s]. Expecting one of: %v", function, cc.functions()))
	}

	return f(stub, args)
}

func (cc *ExampleCC) put(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Invalid args. Expecting key and value")
	}

	key := args[0]
	value := args[1]

	if err := stub.PutState(key, []byte(value)); err != nil {
		return shim.Error(fmt.Sprintf("Error putting data for key [%s]: %s", key, err))
	}

	return shim.Success([]byte(value))
}

func (cc *ExampleCC) get(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Invalid args. Expecting key")
	}

	key := args[0]

	value, err := stub.GetState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error getting data for key [%s]: %s", key, err))
	}

	return shim.Success([]byte(value))
}

func (cc *ExampleCC) del(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Invalid args. Expecting key")
	}

	key := args[0]

	err := stub.DelState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to delete state for [%s]: %s", key, err))
	}

	return shim.Success(nil)
}

func (cc *ExampleCC) putPrivate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("Invalid args. Expecting collection, key and value")
	}

	coll := args[0]
	key := args[1]
	value := args[2]

	if err := stub.PutPrivateData(coll, key, []byte(value)); err != nil {
		return shim.Error(fmt.Sprintf("Error putting private data for collection [%s] and key [%s]: %s", coll, key, err))
	}

	return shim.Success(nil)
}

func (cc *ExampleCC) getPrivate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Invalid args. Expecting collection and key")
	}

	coll := args[0]
	key := args[1]

	value, err := stub.GetPrivateData(coll, key)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error getting private data for collection [%s] and key [%s]: %s", coll, key, err))
	}

	return shim.Success([]byte(value))
}

func (cc *ExampleCC) queryPrivate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Invalid args. Expecting collection and query expression")
	}

	coll := args[0]
	query := strings.Replace(args[1], "`", `"`, -1)
	query = strings.Replace(query, "|", `,`, -1)

	it, err := stub.GetPrivateDataQueryResult(coll, query)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error querying private data for collection [%s] and query [%s]: %s", coll, query, err))
	}
	defer func() {
		if err := it.Close(); err != nil {
			fmt.Printf("Error closing keys iterator: %s\n", err)
		}
	}()

	var results []*queryresult.KV
	for it.HasNext() {
		result, err := it.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("query operation on private data failed. Error accessing state: %s", err))
		}

		results = append(results, result)
	}

	jsonResults, err := json.Marshal(results)
	if err != nil {
		return shim.Error(fmt.Sprintf("query operation on private data failed. Error marshaling JSON: %s", err))
	}

	return shim.Success(jsonResults)
}

func (cc *ExampleCC) getPrivateByRange(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("Invalid args. Expecting collection and keyFrom, keyTo")
	}

	coll := args[0]
	keyFrom := args[1]
	keyTo := args[2]

	it, err := stub.GetPrivateDataByRange(coll, keyFrom, keyTo)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error getting private data by range for collection [%s] and keys [%s to %s]: %s", coll, keyFrom, keyTo, err))
	}

	kvPair := ""
	for it.HasNext() {
		kv, err := it.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("Error getting next value for private data collection [%s]: %s", coll, err))
		}
		kvPair += fmt.Sprintf("%s=%s ", kv.Key, kv.Value)
	}

	return shim.Success([]byte(kvPair))
}

func (cc *ExampleCC) delPrivate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Invalid args. Expecting collection and key")
	}

	coll := args[0]
	key := args[1]

	err := stub.DelPrivateData(coll, key)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error getting private data for collection [%s] and key [%s]: %s", coll, key, err))
	}

	return shim.Success(nil)
}

func (cc *ExampleCC) putPrivateMultiple(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("Invalid args. Expecting collection1, key1, value1, collection2, key2, value2, etc.")
	}

	ckvs, err := asTuples3(args)
	if err != nil {
		return shim.Error(err.Error())
	}

	var keys string
	for _, ckv := range ckvs {
		coll := ckv.v1
		key := ckv.v2
		value := ckv.v3

		if keys == "" {
			keys = key
		} else {
			keys += "," + key
		}

		if value != "" {
			if err := stub.PutPrivateData(coll, key, []byte(value)); err != nil {
				return shim.Error(fmt.Sprintf("Error putting private data for collection [%s] and key [%s]: %s", coll, key, err))
			}
		} else {
			if _, err := stub.GetPrivateData(coll, key); err != nil {
				return shim.Error(fmt.Sprintf("Error getting private data for collection [%s] and key [%s]: %s", coll, key, err))
			}
		}
	}

	return shim.Success(nil)
}

func (cc *ExampleCC) getPrivateMultiple(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("Invalid args. Expecting collection1, key1, collection2, key2, etc.")
	}

	cks, err := asTuples2(args)
	if err != nil {
		return shim.Error(err.Error())
	}

	var values string
	for i, ck := range cks {
		coll := ck.v1
		key := ck.v2

		value, err := stub.GetPrivateData(coll, key)
		if err != nil {
			return shim.Error(fmt.Sprintf("Error getting private data for collection [%s] and key [%s]: %s", coll, key, err))
		}

		if i == 0 {
			values = string(value)
		} else {
			values += "," + string(value)
		}
	}

	return shim.Success([]byte(values))
}

func (cc *ExampleCC) putBoth(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 5 {
		return shim.Error("Invalid args. Expecting key, value, collection, privkey and privvalue")
	}

	key := args[0]
	value := args[1]
	coll := args[2]
	privKey := args[3]
	privValue := args[4]

	if err := stub.PutState(key, []byte(value)); err != nil {
		return shim.Error(fmt.Sprintf("Error putting state for key [%s]: %s", key, err))
	}
	if err := stub.PutPrivateData(coll, privKey, []byte(privValue)); err != nil {
		return shim.Error(fmt.Sprintf("Error putting private data for collection [%s] and key [%s]: %s", coll, key, err))
	}

	return shim.Success(nil)
}

func (cc *ExampleCC) getAndPutBoth(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 5 {
		return shim.Error("Invalid args. Expecting key, value, collection, privkey and privvalue")
	}

	key := args[0]
	value := args[1]
	coll := args[2]
	privKey := args[3]
	privValue := args[4]

	oldValue, err := stub.GetState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error getting state for key [%s]: %s", key, err))
	}
	if oldValue != nil && value != "" {
		value = value + "_" + string(oldValue)
	}

	oldPrivValue, err := stub.GetPrivateData(coll, privKey)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error getting private data for collection [%s] and key [%s]: %s", coll, privKey, err))
	}
	if oldPrivValue != nil && privValue != "" {
		privValue = privValue + "_" + string(oldPrivValue)
	}

	if value != "" {
		if err := stub.PutState(key, []byte(value)); err != nil {
			return shim.Error(fmt.Sprintf("Error putting state for key [%s]: %s", key, err))
		}
	}
	if privValue != "" {
		if err := stub.PutPrivateData(coll, privKey, []byte(privValue)); err != nil {
			return shim.Error(fmt.Sprintf("Error putting private data for collection [%s] and key [%s]: %s", coll, privKey, err))
		}
	}

	return shim.Success(nil)
}

func (cc *ExampleCC) getAndPutPrivate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("Invalid args. Expecting collection, privkey and privvalue")
	}

	coll := args[0]
	privKey := args[1]
	privValue := args[2]

	oldPrivValue, err := stub.GetPrivateData(coll, privKey)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error getting private data for collection [%s] and key [%s]: %s", coll, privKey, err))
	}
	if oldPrivValue != nil {
		privValue = privValue + "_" + string(oldPrivValue)
	}

	if err := stub.PutPrivateData(coll, privKey, []byte(privValue)); err != nil {
		return shim.Error(fmt.Sprintf("Error putting private data for collection [%s] and key [%s]: %s", coll, privKey, err))
	}

	return shim.Success(nil)
}

func (cc *ExampleCC) putCAS(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Invalid args. Expecting collection and value")
	}

	coll := args[0]
	value := []byte(args[1])

	casClient, err := cc.CreateDCASClientStubWrapper(coll, stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error creating DCAS client for collection [%s]: %s", coll, err))
	}

	cID, err := casClient.Put(bytes.NewReader(value))
	if err != nil {
		return shim.Error(fmt.Sprintf("Error putting private data for collection [%s]: %s", coll, err))
	}

	return shim.Success([]byte(cID))
}

func (cc *ExampleCC) putCASMultiple(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("Invalid args. Expecting collection1, value1, collection2, value2, etc.")
	}

	ckvs, err := asTuples2(args)
	if err != nil {
		return shim.Error(err.Error())
	}

	var keys string
	for _, ckv := range ckvs {
		coll := ckv.v1
		value := []byte(ckv.v2)

		casClient, err := cc.CreateDCASClientStubWrapper(coll, stub)
		if err != nil {
			return shim.Error(fmt.Sprintf("Error creating DCAS client for collection [%s]: %s", coll, err))
		}

		cID, err := casClient.Put(bytes.NewReader(value))
		if err != nil {
			return shim.Error(fmt.Sprintf("Error putting CAS data for collection [%s]: %s", coll, err))
		}

		if keys == "" {
			keys = cID
		} else {
			keys += "," + cID
		}
	}

	return shim.Success([]byte(keys))
}

func (cc *ExampleCC) getCAS(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Invalid args. Expecting collection and key")
	}

	coll := args[0]
	cID := args[1]

	casClient, err := cc.CreateDCASClientStubWrapper(coll, stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error creating DCAS client for collection [%s]: %s", coll, err))
	}

	value := bytes.NewBuffer(nil)
	err = casClient.Get(cID, value)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error getting CAS data for collection [%s] and CID [%s]: %s", coll, cID, err))
	}

	return shim.Success(value.Bytes())
}

func (cc *ExampleCC) getAndPutCAS(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Invalid args. Expecting collection and value")
	}

	coll := args[0]
	privBytes := []byte(args[1])

	cID, err := dcas.GetCID(privBytes, dcas.CIDV1, cid.Raw, mh.SHA2_256)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error getting CAS key for [%s]: %s", privBytes, err))
	}

	casClient, err := cc.CreateDCASClientStubWrapper(coll, stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error creating DCAS client for collection [%s]: %s", coll, err))
	}

	oldPrivValue := bytes.NewBuffer(nil)
	err = casClient.Get(cID, oldPrivValue)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error getting DCAS data for collection [%s] and CID [%s]: %s", coll, cID, err))
	}
	if len(oldPrivValue.Bytes()) > 0 {
		return shim.Success([]byte(cID))
	}

	cid2, err := casClient.Put(bytes.NewReader(privBytes))
	if err != nil {
		return shim.Error(fmt.Sprintf("Error putting DCAS data for collection [%s]: %s", coll, err))
	}

	if cid2 != cID {
		return shim.Error(fmt.Sprintf("Unexpected: cid2 [%s] does not match cid2 [%s] for collection [%s]", coll, cID, cid2))
	}

	return shim.Success([]byte(cID))
}

type argStruct struct {
	Args []string `json:"Args"`
}

func asBytes(args []string) [][]byte {
	bytes := make([][]byte, len(args))
	for i, arg := range args {
		bytes[i] = []byte(arg)
	}
	return bytes
}

func (cc *ExampleCC) invokeCC(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error(`Invalid args. Expecting target chaincode, target channel (blank if same channel), and chaincode args in the format {"Args":["arg1","arg2",...]}`)
	}

	ccName := args[0]
	channelID := args[1]
	invokeArgsJSON := strings.Replace(args[2], "`", `"`, -1)
	invokeArgsJSON = strings.Replace(invokeArgsJSON, "|", `,`, -1)

	argStruct := argStruct{}
	if err := json.Unmarshal([]byte(invokeArgsJSON), &argStruct); err != nil {
		return shim.Error(fmt.Sprintf("Invalid invoke args: %s", err))
	}

	return stub.InvokeChaincode(ccName, asBytes(argStruct.Args), channelID)
}

func (cc *ExampleCC) warmup(shim.ChaincodeStubInterface, []string) pb.Response {
	return shim.Success(nil)
}

func (cc *ExampleCC) initRegistry() {
	cc.funcRegistry = make(map[string]invokeFunc)
	cc.funcRegistry[warmupFunc] = cc.warmup
	cc.funcRegistry[getFunc] = cc.get
	cc.funcRegistry[putFunc] = cc.put
	cc.funcRegistry[delFunc] = cc.del
	cc.funcRegistry[getPrivateFunc] = cc.getPrivate
	cc.funcRegistry[queryPrivateFunc] = cc.queryPrivate
	cc.funcRegistry[putPrivateFunc] = cc.putPrivate
	cc.funcRegistry[getPrivateMultipleFunc] = cc.getPrivateMultiple
	cc.funcRegistry[putPrivateMultipleFunc] = cc.putPrivateMultiple
	cc.funcRegistry[delPrivateFunc] = cc.delPrivate
	cc.funcRegistry[getAndPutPrivateFunc] = cc.getAndPutPrivate
	cc.funcRegistry[putBothFunc] = cc.putBoth
	cc.funcRegistry[getAndPutBothFunc] = cc.getAndPutBoth
	cc.funcRegistry[invokeCCFunc] = cc.invokeCC
	cc.funcRegistry[getPrivateByRangeFunc] = cc.getPrivateByRange
	cc.funcRegistry[putCASFunc] = cc.putCAS
	cc.funcRegistry[putCASMultipleFunc] = cc.putCASMultiple
	cc.funcRegistry[getCASFunc] = cc.getCAS
	cc.funcRegistry[getAndPutCASFunc] = cc.getAndPutCAS
}

func (cc *ExampleCC) functions() []string {
	var funcs []string
	for key := range cc.funcRegistry {
		funcs = append(funcs, key)
	}
	return funcs
}

// getCASKey returns the content-addressable key for the given content,
// encoded in base58 so that it may be used as a key in Fabric.
func getCASKey(content []byte) string {
	hash := getHash(content)
	buf := make([]byte, base64.URLEncoding.EncodedLen(len(hash)))
	base64.URLEncoding.Encode(buf, hash)
	return base58.Encode(buf)
}

// getCASKeyAndValue returns the content-addressable key for the given content,
// (encoded in base58 so that it may be used as a key in Fabric) along with the
// normalized value (i.e. if the content is a JSON doc then the fields
// are marshaled in a deterministic order).
func getCASKeyAndValue(content []byte) (string, []byte, error) {
	bytes, err := getNormalizedContent(content)
	if err != nil {
		return "", nil, err
	}
	return getCASKey(bytes), bytes, nil
}

// getNormalizedContent ensures that, if the content is a JSON doc, then the fields are marshaled in a deterministic order.
func getNormalizedContent(content []byte) ([]byte, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal(content, &m)
	if err != nil {
		// This is not a JSON document
		return content, nil
	}

	// This is a JSON doc. Re-marshal it in order to ensure that the JSON fields are marshaled in a deterministic order.
	bytes, err := json.Marshal(&m)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// getHash will compute the hash for the supplied bytes using SHA256
func getHash(bytes []byte) []byte {
	h := crypto.SHA256.New()
	// added no lint directive because there's no error from source code
	// error cannot be produced, checked google source
	h.Write(bytes) //nolint
	return h.Sum(nil)
}

type tuple2 struct {
	v1 string
	v2 string
}

func asTuples2(args []string) ([]*tuple2, error) {
	if len(args) == 0 {
		return nil, nil
	}

	if len(args)%2 != 0 {
		return nil, fmt.Errorf("missing values")
	}

	var tuples []*tuple2
	for i := 0; i < len(args); i = i + 2 {
		tuples = append(tuples, &tuple2{v1: args[i], v2: args[i+1]})
	}
	return tuples, nil
}

type tuple3 struct {
	tuple2
	v3 string
}

func asTuples3(args []string) ([]*tuple3, error) {
	if len(args) == 0 {
		return nil, nil
	}

	if len(args)%3 != 0 {
		return nil, fmt.Errorf("missing values")
	}

	var tuples []*tuple3
	for i := 0; i < len(args); i = i + 3 {
		tuples = append(tuples, &tuple3{tuple2: tuple2{v1: args[i], v2: args[i+1]}, v3: args[i+2]})
	}
	return tuples, nil
}
