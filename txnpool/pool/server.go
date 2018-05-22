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

// Package proc privides functions for handle messages from
// consensus/ledger/net/http/validators
package pool

import (
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	ctypes "github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
	tsend "github.com/ontio/ontology/txnpool/actor/send"
	ttypes "github.com/ontio/ontology/txnpool/types"
	vtypes "github.com/ontio/ontology/validator/types"

	"fmt"
	//"time"
	"time"
)

const (
	MAX_PENDING      = 20000 // The max length of pending txs
	MAX_CHECKING     = 2000  // The length of pending tx from net and http
	CHECKING_TIMEOUT = 30
)

type PoolServer struct {
	txpool       *txpool
	pendingBlock *BlockEntry
	sender       *tsend.Sender
}

func NewTxPoolServer(sender *tsend.Sender) *PoolServer {
	s := &PoolServer{sender: sender}
	s.init()
	return s
}

func (self *PoolServer) init() {
	self.txpool = newTxPool()
	self.pendingBlock = newBlockEntry()
}

func (self *PoolServer) Stop() {
	self.sender.Stop()
}

func (self *PoolServer) verifyFail(hash common.Uint256, err errors.ErrCode) {
	self.txpool.checkingFail(hash)
	self.pendingBlock.updateProcessedTx(hash, err)
}

func (self *PoolServer) verifySuccess(rsp *vtypes.VerifyTxRsp) {
	passed, passedTxEntry := self.txpool.checkPassed(rsp)
	if !passed {
		return
	}
	self.txpool.updateTransaction(passedTxEntry)
	self.txpool.appendVerified(passedTxEntry)

	if passedTxEntry.HttpSender {
		self.sender.SendTxToNetActor(passedTxEntry.Tx)
	}
	self.pendingBlock.updateProcessedTx(passedTxEntry.Tx.Hash(), errors.ErrNoError)
}

func (self *PoolServer) HandleGetVerifiedTxFromPool(hash common.Uint256) *ctypes.Transaction {
	fmt.Println("HandleGetVerifiedTxFromPool==")
	txEntry := self.txpool.getVerifiedTransaction(hash)
	if txEntry == nil {
		return nil
	}
	return txEntry.Tx
}
func (self *PoolServer) HandleVerifyBlockReq(height uint32, txs []*ctypes.Transaction, consusActor *actor.PID) {
	if len(txs) == 0 {
		return
	}
	fmt.Println("HandleVerifyBlockReq=====================", len(txs))
	verified, unverify, reverify := self.txpool.getVerifyBlockTxsState(txs, height)
	self.txpool.removeInvalidInPassed()
	fmt.Println("verifiedTxs:", len(verified))
	fmt.Println("unverify:", len(unverify))
	fmt.Println("pending:", self.txpool.numPending)
	fmt.Println("reverify:", len(reverify))
	for _, tx := range unverify {
		if self.txpool.appendCheckingStateless(tx) {
			self.sender.SendVerifyTxReq(vtypes.Stateless, tx)
		}
	}

	for _, tx := range reverify {
		if self.txpool.appendCheckingStateful(tx) {
			self.sender.SendVerifyTxReq(vtypes.Stateful, tx)
		}
	}
	self.pendingBlock.updateBlock(verified, unverify, reverify, height, txs, consusActor)
	self.txpool.removeInvalidInChecking()
}
func (self *PoolServer) HandleGetTxEntrysFromPool(byCount bool, height uint32) []*ttypes.TxEntry {
	fmt.Println("\nHandleGetTxEntrysFromPool==", height)
	verifiedTxs, reVerifyTxs := self.txpool.takeVerifiedTransactions(byCount, height)
	fmt.Println("verifiedTxs:", len(verifiedTxs))
	self.txpool.removeInvalidInPassed()
	fmt.Println("reVerifyTxs:", len(reVerifyTxs))
	fmt.Println("pending:", self.txpool.numPending)
	fmt.Println("checking:", self.txpool.numChecking)
	for _, v := range reVerifyTxs {
		if self.txpool.appendCheckingStateful(v.Tx) {
			self.sender.SendVerifyTxReq(vtypes.Stateful, v.Tx)
		}
	}
	self.txpool.removeInvalidInChecking()
	fmt.Println("checking:", self.txpool.numChecking)
	return verifiedTxs
}

func (self *PoolServer) HandleSaveBlockComplete(txs []*ctypes.Transaction) {
	tmp := self.txpool.size()
	//hash := txs[0].Hash()
	//fmt.Println(hash.ToArray())
	//fmt.Println(self.txpool.txEntrys)
	self.txpool.removeTransactions(txs)
	fmt.Println("HandleSaveBlockComplete txpool.size:", len(txs), " bef:", tmp, "aft:", self.txpool.size())
}

func (self *PoolServer) HandleGetStatistics() []uint64 {
	return []uint64{}
}

func (self *PoolServer) HandleIsContainTx(hash common.Uint256) bool {
	if self.txpool.getTransaction(hash) != nil {
		return true
	}
	return false
}

func (self *PoolServer) HandleGetTxVerifyResult(hash common.Uint256) *ttypes.TxEntry {
	txEntry := self.txpool.getVerifiedTransaction(hash)
	if txEntry == nil {
		return nil
	}
	return txEntry
}

func (self *PoolServer) HandleAppendTxReq(sender bool, tx *ctypes.Transaction) {

	if txEntry := self.txpool.putTransaction(tx, sender); txEntry != nil {
		if self.txpool.numPending < MAX_PENDING {
			self.txpool.appendPending(txEntry)
		}
	}
}

func (self *PoolServer) HandleVerifyTxRsp(rsp *vtypes.VerifyTxRsp) {
	if rsp == nil {
		return
	}
	if rsp.ErrCode != errors.ErrNoError {
		log.Info(fmt.Sprintf("%d: Transaction %x invalid: %s", rsp.VerifyType, rsp.Hash, rsp.ErrCode.Error()))
		self.verifyFail(rsp.Hash, rsp.ErrCode)
		return
	}
	self.verifySuccess(rsp)
}

func (self *PoolServer) HandleUpdatePool() {

	now := time.Now().Unix()

	for _, v := range self.txpool.checking {
		txEntry := v
		if txEntry.Stage == ttypes.Invalid {
			continue
		}
		if now-txEntry.TimeStamp < CHECKING_TIMEOUT {
			break
		}
		fmt.Println("time:", now-txEntry.TimeStamp)
		txEntry.Stage = ttypes.Invalid
		newEntry := &ttypes.TxEntry{
			txEntry.Tx,
			txEntry.Gas,
			txEntry.PassStateful,
			txEntry.PassStateless,
			ttypes.Checking,
			txEntry.HttpSender,
			time.Now().Unix(),
			txEntry.VerifyHeight,
		}
		self.txpool.txEntrys[txEntry.Tx.Hash()] = newEntry
		if self.txpool.appendCheckingStateless(txEntry.Tx) {
			self.sender.SendVerifyTxReq(vtypes.Stateless, txEntry.Tx)
		}

	}

	if self.txpool.numChecking <= (MAX_CHECKING/4*3) && self.txpool.numPending > 0 {
		num := self.txpool.numPending
		if num > MAX_CHECKING/4 {
			num = MAX_CHECKING / 4
		}
		fmt.Println("======HandleUpdatePool====take pending====", num)
		for _, v := range self.txpool.pending[:num] {
			v.Stage = ttypes.Invalid
			newEntry := &ttypes.TxEntry{
				v.Tx,
				v.Gas,
				v.PassStateful,
				v.PassStateless,
				ttypes.Checking,
				v.HttpSender,
				time.Now().Unix(),
				v.VerifyHeight,
			}
			self.txpool.txEntrys[newEntry.Tx.Hash()] = newEntry
			if self.txpool.appendCheckingStateless(newEntry.Tx) {
				self.txpool.numPending--
				self.sender.SendVerifyTxReq(vtypes.Stateless, newEntry.Tx)
			}
		}
		self.txpool.removeInvalidInPending()
	}

}
