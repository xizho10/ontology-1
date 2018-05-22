/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */
package types

import (
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
	//vtypes "github.com/ontio/ontology/validator/types"
)

// ActorType enumerates the kind of actor
type ActorType uint8

const (
	_           ActorType = iota
	TxPoolActor           // Actor that handles consensus msg
	NetActor              // Actor to send msg to the net actor
	MaxActor
)

type VerifyStage int

const (
	_ VerifyStage = iota
	UnVerify
	PassStateless
	PassStateful
	Verified
	Invalid
)

// VerifyTxResult returns a single transaction's verified result.
type TxVerifyResult struct {
	Tx *types.Transaction
	VerifyResult
}
type VerifyResult struct {
	Height  uint32         // The height in which tx was verified
	ErrCode errors.ErrCode // Verified result
}

type TxEntry struct {
	Tx  *types.Transaction // transaction which has been verified
	Gas uint64             // Total fee per transaction
	//StatefulResult  *VerifyResult      // the result from each type validator
	//StatelessResult *VerifyResult
	PassStateful  bool
	PassStateless bool
	Stage         VerifyStage
	HttpSender    bool
	TimeStamp     int64
	VerifyHeight  uint32
}

// TxReq specifies the api that how to submit a new transaction.
// Input: transacton and submitter type
type AppendTxReq struct {
	Tx         *types.Transaction
	HttpSender bool
}

// TxRsp returns the result of submitting tx, including
// a transaction hash and error code.
type AppendTxRsp struct {
	Hash    common.Uint256
	ErrCode errors.ErrCode
}

// GetTxnReq specifies the api that how to get the transaction.
// Input: a transaction hash
type GetVerifiedTxFromPoolReq struct {
	Hash common.Uint256
}

// GetTxnRsp returns a transaction for the specified tx hash.
type GetVerifiedTxFromPoolRsp struct {
	Txn *types.Transaction
}

// CheckTxnReq specifies the api that how to check whether a
// transaction in the pool.
// Input: a transaction hash
type IsTxInPoolReq struct {
	Hash common.Uint256
}

// CheckTxnRsp returns a value for the CheckTxnReq, if the
// transaction in the pool, value is true, or false.
type IsTxInPoolRsp struct {
	Ok bool
}

// GetTxnStatusReq specifies the api that how to get a transaction
// status.
// Input: a transaction hash.
type GetTxVerifyResultReq struct {
	Hash common.Uint256
}

// GetTxnStatusRsp returns a transaction status for GetTxnStatusReq.
// Output: a transaction hash and it's verified result.
type GetTxVerifyResultRsp struct {
	Hash    common.Uint256
	TxEntry *TxEntry
}

// GetTxnStats specifies the api that how to get the tx statistics.
type GetTxVerifyResultStaticsReq struct {
}

// GetTxnStatsRso returns the tx statistics.
type GetTxVerifyResultStaticsRsp struct {
	Count []uint64
}

// consensus messages
// GetTxnPoolReq specifies the api that how to get the valid transaction list.
type GetVerifiedTxnFromPoolReq struct {
	ByCount bool
	Height  uint32
}

// GetTxnPoolRsp returns a transaction list for GetTxnPoolReq.
type GetVerifiedTxnFromPoolRsp struct {
	TxnPool []*TxEntry
}

// VerifyBlockReq specifies that api that how to verify a block from consensus.
type VerifyBlockReq struct {
	Height uint32
	Txs    []*types.Transaction
}

// VerifyBlockRsp returns a verified result for VerifyBlockReq.
type VerifyBlockRsp struct {
	TxResults []*TxVerifyResult
}
