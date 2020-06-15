/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

// ProposalResponsesBuilder builds a slice of mock ProposalResponse's
type ProposalResponsesBuilder struct {
	responses []*ProposalResponseBuilder
}

// NewProposalResponsesBuilder returns a mock proposal responses builder
func NewProposalResponsesBuilder() *ProposalResponsesBuilder {
	return &ProposalResponsesBuilder{}
}

// ProposalResponse adds a new proposal response
func (b *ProposalResponsesBuilder) ProposalResponse() *ProposalResponseBuilder {
	responseBuilder := NewProposalResponseBuilder()
	b.responses = append(b.responses, responseBuilder)
	return responseBuilder
}

// Build returns a slice of mock ProposalResponse's
func (b *ProposalResponsesBuilder) Build() []*pb.ProposalResponse {
	responses := make([]*pb.ProposalResponse, len(b.responses))
	for i, rb := range b.responses {
		responses[i] = rb.Build()
	}
	return responses
}

// ProposalResponseBuilder builds a mock ProposalResponse
type ProposalResponseBuilder struct {
	version     int32
	timestamp   *timestamp.Timestamp
	response    *ResponseBuilder
	payload     *ResponsePayloadBuilder
	endorsement *EndorsementBuilder
}

// NewProposalResponseBuilder returns a mock proposal response builder
func NewProposalResponseBuilder() *ProposalResponseBuilder {
	return &ProposalResponseBuilder{}
}

// Version sets the version
func (b *ProposalResponseBuilder) Version(value int32) *ProposalResponseBuilder {
	b.version = value
	return b
}

// Timestamp sets the timestamp
func (b *ProposalResponseBuilder) Timestamp(value *timestamp.Timestamp) *ProposalResponseBuilder {
	b.timestamp = value
	return b
}

// Response returns a new response builder
func (b *ProposalResponseBuilder) Response() *ResponseBuilder {
	b.response = NewResponseBuilder()
	return b.response
}

// Payload returns a new response payload builder
func (b *ProposalResponseBuilder) Payload() *ResponsePayloadBuilder {
	b.payload = NewResponsePayloadBuilder()
	return b.payload
}

// Endorsement returns a new endorsement builder
func (b *ProposalResponseBuilder) Endorsement() *EndorsementBuilder {
	b.endorsement = NewEndorsementBuilder()
	return b.endorsement
}

// Build returns a proposal response
func (b *ProposalResponseBuilder) Build() *pb.ProposalResponse {
	pr := &pb.ProposalResponse{
		Version:   b.version,
		Timestamp: b.timestamp,
	}
	if b.response != nil {
		pr.Response = b.response.Build()
	}
	if b.payload != nil {
		pr.Payload = b.payload.BuildBytes()
	}
	if b.endorsement != nil {
		pr.Endorsement = b.endorsement.Build()
	}
	return pr
}

// ResponseBuilder builds a mock Response
type ResponseBuilder struct {
	response *pb.Response
}

// NewResponseBuilder returns a mock response builder
func NewResponseBuilder() *ResponseBuilder {
	return &ResponseBuilder{
		response: &pb.Response{},
	}
}

// Message sets the response message
func (b *ResponseBuilder) Message(value string) *ResponseBuilder {
	b.response.Message = value
	return b
}

// Status sets the response status code
func (b *ResponseBuilder) Status(value int32) *ResponseBuilder {
	b.response.Status = value
	return b
}

// Payload sets the response payload
func (b *ResponseBuilder) Payload(value []byte) *ResponseBuilder {
	b.response.Payload = value
	return b
}

// Build returns the response
func (b *ResponseBuilder) Build() *pb.Response {
	return b.response
}

// ResponsePayloadBuilder builds a mock Proposal Response Payload
type ResponsePayloadBuilder struct {
	proposalHash    []byte
	chaincodeAction *ChaincodeActionBuilder
}

// NewResponsePayloadBuilder returns a mock Proposal Response Payload builder
func NewResponsePayloadBuilder() *ResponsePayloadBuilder {
	return &ResponsePayloadBuilder{}
}

// ProposalHash sets the proposal hash
func (b *ResponsePayloadBuilder) ProposalHash(value []byte) *ResponsePayloadBuilder {
	b.proposalHash = value
	return b
}

// ChaincodeAction returns a ChaincodeActionBuilder
func (b *ResponsePayloadBuilder) ChaincodeAction() *ChaincodeActionBuilder {
	b.chaincodeAction = NewChaincodeActionBuilder()
	return b.chaincodeAction
}

// Build returns the proposal response payload
func (b *ResponsePayloadBuilder) Build() *pb.ProposalResponsePayload {
	prp := &pb.ProposalResponsePayload{}
	if b.chaincodeAction != nil {
		ca := b.chaincodeAction.Build()
		extension, err := proto.Marshal(ca)
		if err != nil {
			panic(err.Error())
		}
		prp.Extension = extension
	}
	return prp
}

// BuildBytes returns the proposal response payload bytes
func (b *ResponsePayloadBuilder) BuildBytes() []byte {
	prp := b.Build()
	prpBytes, err := proto.Marshal(prp)
	if err != nil {
		panic(err.Error())
	}
	return prpBytes
}

// ChaincodeActionBuilder builds a mock ChaincodeAction
type ChaincodeActionBuilder struct {
	chaincodeAction *pb.ChaincodeAction
	events          *ChaincodeEventBuilder
	results         *mocks.ReadWriteSetBuilder
}

// NewChaincodeActionBuilder returns a mock ChaincodeAction builder
func NewChaincodeActionBuilder() *ChaincodeActionBuilder {
	return &ChaincodeActionBuilder{
		chaincodeAction: &pb.ChaincodeAction{},
	}
}

// ChaincodeID sets the chaincode ID
func (b *ChaincodeActionBuilder) ChaincodeID(name, path, version string) *ChaincodeActionBuilder {
	b.chaincodeAction.ChaincodeId = &pb.ChaincodeID{
		Name:    name,
		Path:    path,
		Version: version,
	}
	return b
}

// Results returns a read-write set builder
func (b *ChaincodeActionBuilder) Results() *mocks.ReadWriteSetBuilder {
	b.results = mocks.NewReadWriteSetBuilder()
	return b.results
}

// Response sets the response
func (b *ChaincodeActionBuilder) Response(status int32, payload []byte, message string) *ChaincodeActionBuilder {
	b.chaincodeAction.Response = &pb.Response{
		Status:  status,
		Payload: payload,
		Message: message,
	}
	return b
}

// Events sets the results
func (b *ChaincodeActionBuilder) Events() *ChaincodeEventBuilder {
	b.events = NewChaincodeEventBuilder()
	return b.events
}

// Build returns the ChaincodeAction
func (b *ChaincodeActionBuilder) Build() *pb.ChaincodeAction {
	if b.events != nil {
		b.chaincodeAction.Events = b.events.BuildBytes()
	}
	if b.results != nil {
		b.chaincodeAction.Results = b.results.BuildBytes()
	}
	return b.chaincodeAction
}

// ChaincodeEventBuilder builds a mock ChaincodeAction
type ChaincodeEventBuilder struct {
	event *pb.ChaincodeEvent
}

// NewChaincodeEventBuilder returns a mock ChaincodeAction builder
func NewChaincodeEventBuilder() *ChaincodeEventBuilder {
	return &ChaincodeEventBuilder{
		event: &pb.ChaincodeEvent{},
	}
}

// ChaincodeID sets the chaincode ID
func (b *ChaincodeEventBuilder) ChaincodeID(value string) *ChaincodeEventBuilder {
	b.event.ChaincodeId = value
	return b
}

// EventName sets the event name
func (b *ChaincodeEventBuilder) EventName(value string) *ChaincodeEventBuilder {
	b.event.EventName = value
	return b
}

// TxID sets the transaction ID
func (b *ChaincodeEventBuilder) TxID(value string) *ChaincodeEventBuilder {
	b.event.TxId = value
	return b
}

// Payload sets the payload
func (b *ChaincodeEventBuilder) Payload(value []byte) *ChaincodeEventBuilder {
	b.event.Payload = value
	return b
}

// Build returns the ChaincodeEvent
func (b *ChaincodeEventBuilder) Build() *pb.ChaincodeEvent {
	return b.event
}

// BuildBytes returns the ChaincodeEvent bytes
func (b *ChaincodeEventBuilder) BuildBytes() []byte {
	bytes, err := proto.Marshal(b.Build())
	if err != nil {
		panic(err.Error())
	}
	return bytes
}

// EndorsementBuilder builds a mock ChaincodeAction
type EndorsementBuilder struct {
	endorsement *pb.Endorsement
}

// NewEndorsementBuilder returns a mock ChaincodeAction builder
func NewEndorsementBuilder() *EndorsementBuilder {
	return &EndorsementBuilder{
		endorsement: &pb.Endorsement{},
	}
}

// Endorser sets the endorser
func (b *EndorsementBuilder) Endorser(value []byte) *EndorsementBuilder {
	b.endorsement.Endorser = value
	return b
}

// Signature sets the signature
func (b *EndorsementBuilder) Signature(value []byte) *EndorsementBuilder {
	b.endorsement.Signature = value
	return b
}

// Build returns the Endorsement
func (b *EndorsementBuilder) Build() *pb.Endorsement {
	return b.endorsement
}
