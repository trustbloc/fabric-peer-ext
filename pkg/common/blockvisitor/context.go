/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockvisitor

import (
	"fmt"
)

// Category defines the error category
type Category string

const (
	// UnmarshalErr indicates that an error occurred while attempting to unmarshal block data
	UnmarshalErr Category = "UNMARSHAL_ERROR"

	// ReadHandlerErr indicates that the read handler returned an error
	ReadHandlerErr Category = "READ_HANDLER_ERROR"

	// WriteHandlerErr indicates that the write handler returned an error
	WriteHandlerErr Category = "WRITE_HANDLER_ERROR"

	// CollHashReadHandlerErr indicates that the collection hash read handler returned an error
	CollHashReadHandlerErr Category = "COLL_HASH_READ_HANDLER_ERROR"

	// CollHashWriteHandlerErr indicates that the collection hash write handler returned an error
	CollHashWriteHandlerErr Category = "COLL_HASH_WRITE_HANDLER_ERROR"

	// LSCCWriteHandlerErr indicates that the LSCC write handler returned an error
	LSCCWriteHandlerErr Category = "LSCCWRITE_HANDLER_ERROR"

	// CCEventHandlerErr indicates that the chaincode event handler returned an error
	CCEventHandlerErr Category = "CCEVENT_HANDLER_ERROR"

	// ConfigUpdateHandlerErr indicates that the configuration update handler returned an error
	ConfigUpdateHandlerErr Category = "CONFIG_UPDATE_HANDLER_ERROR"
)

// Context holds the context at the time of a Visitor error
type Context struct {
	Category      Category
	ChannelID     string
	BlockNum      uint64
	TxNum         uint64
	TxID          string
	Read          *Read
	Write         *Write
	CollHashRead  *CollHashRead
	CollHashWrite *CollHashWrite
	CCEvent       *CCEvent
	LSCCWrite     *LSCCWrite
	ConfigUpdate  *ConfigUpdate
}

func newContext(category Category, channelID string, blockNum uint64, opts ...ctxOpt) *Context {
	ctx := &Context{
		Category:  category,
		ChannelID: channelID,
		BlockNum:  blockNum,
	}

	for _, opt := range opts {
		opt(ctx)
	}

	return ctx
}

// String returns the string representation of Context
func (c *Context) String() string {
	s := fmt.Sprintf("ChannelID: %s, Category: %s, Block: %d, TxNum: %d", c.ChannelID, c.Category, c.BlockNum, c.TxNum)

	if c.TxID != "" {
		s += fmt.Sprintf(", TxID: %s", c.TxID)
	}

	if c.Read != nil {
		s += fmt.Sprintf(", Read: %+v", c.Read)
	}

	if c.Write != nil {
		s += fmt.Sprintf(", Write: %+v", c.Write)
	}

	if c.CCEvent != nil {
		s += fmt.Sprintf(", CCEvent: %+v", c.CCEvent)
	}

	if c.LSCCWrite != nil {
		s += fmt.Sprintf(", LSCCWrite: %+v", c.LSCCWrite)
	}

	if c.ConfigUpdate != nil {
		s += fmt.Sprintf(", ConfigUpdate: %+v", c.ConfigUpdate)
	}

	return s
}

type ctxOpt func(ctx *Context)

func withTxID(txID string) ctxOpt {
	return func(ctx *Context) {
		ctx.TxID = txID
	}
}

func withTxNum(num uint64) ctxOpt {
	return func(ctx *Context) {
		ctx.TxNum = num
	}
}

func withRead(r *Read) ctxOpt {
	return func(ctx *Context) {
		ctx.Read = r
	}
}

func withWrite(w *Write) ctxOpt {
	return func(ctx *Context) {
		ctx.Write = w
	}
}

func withCollHashRead(r *CollHashRead) ctxOpt {
	return func(ctx *Context) {
		ctx.CollHashRead = r
	}
}

func withCollHashWrite(w *CollHashWrite) ctxOpt {
	return func(ctx *Context) {
		ctx.CollHashWrite = w
	}
}

func withLSCCWrite(w *LSCCWrite) ctxOpt {
	return func(ctx *Context) {
		ctx.LSCCWrite = w
	}
}

func withCCEvent(ccEvent *CCEvent) ctxOpt {
	return func(ctx *Context) {
		ctx.CCEvent = ccEvent
	}
}

func withConfigUpdate(u *ConfigUpdate) ctxOpt {
	return func(ctx *Context) {
		ctx.ConfigUpdate = u
	}
}
