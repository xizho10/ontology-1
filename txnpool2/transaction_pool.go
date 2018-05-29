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

package txnpool2

import (
	"errors"
	"time"

	"github.com/ontio/ontology-eventbus/actor"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/types"
	ontErr "github.com/ontio/ontology/errors"
	vt "github.com/ontio/ontology/validator2/types"
)

var TooManyPendingTxError = errors.New("too many pending tx")
var OutofCapacityError = errors.New("out of capacity")
var DuplicateTxError = errors.New("duplicated tx")
var NotEnoughValidatorError = errors.New("not enough validator")

type PoolConfig struct {
	MaxTxInBlock  int
	MaxPendingTx  int
	MaxCheckingTx int
	MaxTxCapacity int
}

type VerifyStage int

const (
	Pending  VerifyStage = 0
	Checking VerifyStage = 1
	Passed   VerifyStage = 2
	Invalid  VerifyStage = 3
)

type TxEntry struct {
	Tx            *types.Transaction // transaction which has been verified
	Fee           common.Fixed64     // Total fee per transaction
	TimeStamp     int64
	VerifyHeight  uint32
	PassStateless bool
	PassStateful  bool

	InBlock bool // is this tx in block check list
	Stage   VerifyStage
}

type BlockEntry struct {
	Txs         []*TxEntry
	NumPassed   int
	BlockHeight uint32
	Sender      *actor.PID
}

type TxPool struct {
	config PoolConfig

	txs          map[common.Uint256]*TxEntry //
	passed       []*TxEntry
	checking     []*TxEntry
	pending      []*TxEntry
	pendingBlock BlockEntry
	numPending   int
	numChecking  int
	numPassed    int
	//passed map[common.Uint256]*txpc.TXEntry // Transactions which have been verified
	//waiting map[common.Uint256]*txpc.TXEntry // Transactions which have scheduled and wait for response
	//pending []*types.Transaction   // Transactions which have not been scheduled to verify yet
	validators [2]struct { // 1: stateless, 2: stateful
		cursor    int
		validator []*vt.RegisterValidatorReq
	}
}

func (self *TxPool) compress() {

}

func (self *TxPool) haveEnoughValidator() bool {
	return len(self.validators[0].validator) > 0 && len(self.validators[1].validator) > 0
}

func (self *TxPool) verifyStateless(entry *TxEntry) {

}

func (self *TxPool) verifyTx(entry *TxEntry) {
	if entry.PassStateless == false {
		//self.validators[0].validator[0].Sender
		panic("unimplemented")
	}
	panic("unimplemented")
}

func (self *TxPool) handleVerifyBlockError() {
	for _, tx := range self.pendingBlock.Txs {
		tx.InBlock = false
	}

	//todo self.pendingBlock.Sender.Tell()

	self.pendingBlock = BlockEntry{}
}

func (self *TxPool) handleVerifyBlockComplete() {
	for _, tx := range self.pendingBlock.Txs {
		tx.InBlock = false
	}

	//todo self.pendingBlock.Sender.Tell()

	self.pendingBlock = BlockEntry{}
}

func (self *TxPool) addStageNum(stage VerifyStage, val int) {
	if stage == Passed {
		self.numPassed += val
	} else if stage == Pending {
		self.numPending += val
	} else if stage == Checking {
		self.numChecking += val
	}
}

func (self *TxPool) handleVerifyResponse(rep *vt.VerifyTxRsp) {
	hash := rep.Hash
	tx, ok := self.txs[hash]
	if ok == false {
		return
	}

	if rep.ErrCode != ontErr.ErrNoError {
		self.addStageNum(tx.Stage, -1)
		tx.Stage = Invalid

		if tx.InBlock {
			self.handleVerifyBlockError()
		}

		delete(self.txs, tx.Tx.Hash())
		return
	}

	if rep.VerifyType == vt.Stateless {
		tx.PassStateless = true
	} else {
		tx.PassStateful = true
		tx.VerifyHeight = rep.Height
	}

	if tx.PassStateful && tx.PassStateless {
		self.addStageNum(tx.Stage, -1)
		tmp := *tx
		txn := &tmp
		tx.Stage = Invalid
		txn.Stage = Passed
		self.addStageNum(Passed, 1)
		self.passed = append(self.passed, txn)
		self.txs[txn.Tx.Hash()] = txn

		if txn.InBlock {
			self.pendingBlock.NumPassed += 1
			if self.pendingBlock.NumPassed == len(self.pendingBlock.Txs) {
				self.handleVerifyBlockComplete()
			}
		}
	}
}

func (self *TxPool) handleVerifyTransaction(tx *types.Transaction) error {
	if self.haveEnoughValidator() == false {
		return NotEnoughValidatorError
	}
	if self.numPending >= self.config.MaxPendingTx {
		return TooManyPendingTxError
	}
	if len(self.txs) >= self.config.MaxTxCapacity {
		return OutofCapacityError
	}
	if self.txs[tx.Hash()] != nil {
		return DuplicateTxError
	}

	entry := &TxEntry{
		Tx: tx,
	}

	self.txs[tx.Hash()] = entry
	if self.numChecking < self.config.MaxCheckingTx {
		entry.TimeStamp = time.Now().Unix()
		entry.Stage = Checking
		self.numChecking += 1
		self.checking = append(self.checking, entry)
		self.verifyTx(entry)
	} else {
		entry.Stage = Pending
		self.numPending += 1
		self.pending = append(self.pending, entry)
	}

	return nil
}

func isValidationExpired(entry *TxEntry, height uint32) bool {
	return entry.VerifyHeight < height
}

func (self *TxPool) GetVerifiedTxs(byCount bool, height uint32) []*TxEntry {
	//invarance: [0, i) passed, [i, p) invalid, [j+1, n) expired, [p, j] not checked yet
	i, p, j := 0, 0, len(self.passed)-1
	for p <= j {
		entry := self.passed[p]
		if entry.Stage == Invalid {
			p += 1
			continue
		} else if isValidationExpired(entry, height) {
			entry.PassStateful = false
			entry.Stage = Pending
			self.passed[p], self.passed[j] = self.passed[j], entry
			j -= 1
		} else {
			self.passed[i], self.passed[p] = entry, self.passed[i]
			p += 1
			i += 1
		}
	}

	// now passed[j+1:] is expired
	log.Infof("transaction pool: exipred %d transactions", len(self.passed)-j-1)
	var expired []*TxEntry
	expired = append(expired, self.passed[j+1:]...)
	self.numPending += len(expired)
	self.pending = append(expired, self.pending...)
	self.passed = self.passed[:i]
	self.numPassed = len(self.passed)

	count := self.config.MaxTxInBlock
	if len(self.passed) < count || !byCount {
		count = len(self.passed)
	}

	// todo : TxEntry may be changed in TxPool, need copy TxEntry object, not only pointer
	txList := make([]*TxEntry, count)
	copy(txList, self.passed)

	return txList
}
