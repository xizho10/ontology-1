package pool

import (
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/common/log"
	ctypes "github.com/ontio/ontology/core/types"
	ttypes "github.com/ontio/ontology/txnpool/types"
	vtypes "github.com/ontio/ontology/validator/types"
	"time"
	//"fmt"
	//"fmt"
	//"os"
)

type txpool struct {
	txEntrys    map[common.Uint256]*ttypes.TxEntry
	passed      []*ttypes.TxEntry
	checking    []*ttypes.TxEntry
	pending     []*ttypes.TxEntry
	numPending  int
	numChecking int
	numPassed   int
}

func newTxPool() *txpool {
	pool := &txpool{
		txEntrys: make(map[common.Uint256]*ttypes.TxEntry),
	}
	return pool
}

func (self *txpool) getTransaction(hash common.Uint256) *ttypes.TxEntry {
	return self.txEntrys[hash]
}

func (self *txpool) checkingFail(hash common.Uint256) {
	txEntry := self.txEntrys[hash]
	txEntry.Stage = ttypes.Invalid
	self.removeTransaction(hash)
	self.numChecking--
}

func (self *txpool) checkPassed(rsp *vtypes.VerifyTxRsp) (bool, *ttypes.TxEntry) {
	txEntry := self.txEntrys[rsp.Hash]
	if txEntry == nil {
		return false, nil
	}
	if !txEntry.PassStateless && rsp.VerifyType == vtypes.Stateless {
		txEntry.PassStateless = true
	} else if !txEntry.PassStateful && rsp.VerifyType == vtypes.Stateful {
		txEntry.PassStateful = true
		txEntry.VerifyHeight = rsp.Height
	}

	if txEntry.PassStateless && txEntry.PassStateful {
		txEntry.Stage = ttypes.Passed
	} else {
		return false, nil
	}

	txEntry.PassStateful = true
	txEntry.PassStateless = true
	//txEntry.Fee = txEntry.Tx.GetTotalFee()
	txEntry.Stage = ttypes.Invalid
	self.numChecking--

	verifiedEntry := &ttypes.TxEntry{
		txEntry.Tx,
		txEntry.Gas,
		txEntry.PassStateful,
		txEntry.PassStateless,
		ttypes.Passed,
		txEntry.HttpSender,
		0,
		txEntry.VerifyHeight,
	}

	return true, verifiedEntry
}

// removePendingTx removes a transaction from the pending list
// when it is handled. And if the submitter of the valid transaction
// is from http, broadcast it to the network. Meanwhile, check if it
// is in the block from consensus.
func (self *txpool) removeTransaction(hash common.Uint256) {
	delete(self.txEntrys, hash)
}

func (self *txpool) removeTransactions(txs []*ctypes.Transaction) {
	//for _, v := range self.txEntrys {
	//	fmt.Println(v.Tx.Hash())
	//}
	//fmt.Println()
	for _, v := range txs {
		//fmt.Println(v.Hash())
		delete(self.txEntrys, v.Hash())
	}
}
func (self *txpool) updateTransaction(txEntry *ttypes.TxEntry) {
	self.txEntrys[txEntry.Tx.Hash()] = txEntry
}

func (self *txpool) putTransaction(tx *ctypes.Transaction, httpSender bool) *ttypes.TxEntry {

	if t := self.txEntrys[tx.Hash()]; t != nil {
		log.Error(" already in pool")
		return nil
	}

	ptx := &ttypes.TxEntry{
		Tx:            tx,
		HttpSender:    httpSender,
		PassStateful:  false,
		PassStateless: false,
		Stage:         ttypes.Pending,
	}
	self.txEntrys[tx.Hash()] = ptx
	return ptx
}

func (self *txpool) size() int {
	return len(self.txEntrys)
}

func (self *txpool) getVerifyBlockTxsState(txs []*ctypes.Transaction,
	height uint32) ([]*ttypes.TxVerifyResult, []*ctypes.Transaction, []*ctypes.Transaction) {

	verifiedTxs := make([]*ttypes.TxVerifyResult, 0, len(txs))
	unVerifiedTxs := make([]*ctypes.Transaction, 0)
	reVerifyTxs := make([]*ctypes.Transaction, 0)

	for _, tx := range txs {
		txEntry := self.txEntrys[tx.Hash()]
		if txEntry == nil {
			continue
		}
		if txEntry.Stage < ttypes.Passed {
			unVerifiedTxs = append(unVerifiedTxs, tx)
			continue
		}
		if txEntry.VerifyHeight < height {
			txEntry.PassStateless = true
			txEntry.PassStateful = false
			txEntry.Stage = ttypes.Invalid
			self.numPassed--
			reVerifyTxs = append(reVerifyTxs, txEntry.Tx)
			continue
		}
		txEntry.Stage = ttypes.Invalid
		self.numPassed--
		verifiedTxs = append(verifiedTxs, &ttypes.TxVerifyResult{txEntry.Tx, ttypes.VerifyResult{txEntry.VerifyHeight, 0}})

	}
	return verifiedTxs, unVerifiedTxs, reVerifyTxs
}

func (self *txpool) getVerifiedTransaction(hash common.Uint256) *ttypes.TxEntry {
	txEntry := self.txEntrys[hash]
	if txEntry == nil {
		return nil
	}
	if txEntry.Stage != ttypes.Passed {
		return nil
	}
	newEntry := &ttypes.TxEntry{
		txEntry.Tx,
		txEntry.Gas,
		txEntry.PassStateful,
		txEntry.PassStateless,
		txEntry.Stage,
		txEntry.HttpSender,
		0,
		txEntry.VerifyHeight,
	}
	return newEntry
}
func (self *txpool) appendPending(txEntry *ttypes.TxEntry) {
	self.numPending++
	self.pending = append(self.pending, txEntry)
}
func (self *txpool) appendVerified(txEntry *ttypes.TxEntry) {
	self.numPassed++
	self.passed = append(self.passed, txEntry)
}
func (self *txpool) appendCheckingStateless(tx *ctypes.Transaction) bool {
	if tx == nil {
		return false
	}
	txEntry := self.getTransaction(tx.Hash())
	if txEntry == nil {
		txEntry = self.putTransaction(tx, false)
	}
	txEntry.PassStateless = false
	txEntry.PassStateful = false
	txEntry.Stage = ttypes.Checking
	txEntry.TimeStamp = time.Now().Unix()
	self.checking = append(self.checking, txEntry)
	self.numChecking++
	return true
}
func (self *txpool) appendCheckingStateful(tx *ctypes.Transaction) bool {
	if tx == nil {
		return false
	}
	txEntry := self.getTransaction(tx.Hash())
	if txEntry == nil {
		return false
	}
	newEntry := &ttypes.TxEntry{
		txEntry.Tx,
		txEntry.Gas,
		false,
		true,
		ttypes.Checking,
		txEntry.HttpSender,
		time.Now().Unix(),
		txEntry.VerifyHeight,
	}
	self.txEntrys[tx.Hash()] = newEntry
	self.checking = append(self.checking, txEntry)
	self.numChecking++
	return true
}
func (self *txpool) getVerifiedCount() int {
	return self.numPassed
}
func (self *txpool) takeVerifiedTransactions(byCount bool, height uint32) ([]*ttypes.TxEntry, []*ttypes.TxEntry) {

	count := int(config.DefConfig.Consensus.MaxTxInBlock)
	if count <= 0 {
		byCount = false
	}
	if self.numPassed < count || !byCount {
		count = self.numPassed
	}
	var num int
	txList := make([]*ttypes.TxEntry, 0, count)
	reVerifyTxs := make([]*ttypes.TxEntry, 0)
	for _, v := range self.passed {
		txEntry := v
		if txEntry.VerifyHeight < height {
			txEntry.PassStateless = true
			txEntry.PassStateful = false
			txEntry.Stage = ttypes.Invalid
			self.numPassed--
			newEntry := &ttypes.TxEntry{
				txEntry.Tx,
				txEntry.Gas,
				false,
				true,
				ttypes.Checking,
				txEntry.HttpSender,
				0,
				txEntry.VerifyHeight,
			}
			reVerifyTxs = append(reVerifyTxs, newEntry)
			continue
		}

		txEntry.Stage = ttypes.Invalid
		self.numPassed--
		txList = append(txList, txEntry)
		num++
		if num >= count {
			break
		}
	}

	return txList, reVerifyTxs
}

func (self *txpool) removeInvalidInPassed() {

	tmp := []*ttypes.TxEntry{}
	for _, v := range self.passed {
		if v.Stage != ttypes.Invalid {
			tmp = append(tmp, v)
		}
	}
	self.passed = tmp
}
func (self *txpool) removeInvalidInChecking() {

	tmp := []*ttypes.TxEntry{}
	for _, v := range self.checking {
		if v.Stage != ttypes.Invalid {
			tmp = append(tmp, v)
		}
	}
	self.checking = tmp
}
func (self *txpool) removeInvalidInPending() {
	tmp := []*ttypes.TxEntry{}
	for _, v := range self.pending {
		if v.Stage != ttypes.Invalid {
			tmp = append(tmp, v)
		}

	}
	self.pending = tmp
}
