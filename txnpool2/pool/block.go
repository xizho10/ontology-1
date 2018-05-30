package pool

import (
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/ontio/ontology/common"
	ctypes "github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
	txpoolcommon "github.com/ontio/ontology/txnpool/common"
	ttypes "github.com/ontio/ontology/txnpool2/types"
)

type BlockEntry struct {
	consusActor    *actor.PID                                // Consensus PID
	height         uint32                                    // The block height
	processedTxs   map[common.Uint256]*ttypes.TxVerifyResult // Transaction which has been processed
	unProcessedTxs map[common.Uint256]*ctypes.Transaction    // Transaction which is not processed
}

func newBlockEntry() *BlockEntry {
	blk := &BlockEntry{
		processedTxs:   make(map[common.Uint256]*ttypes.TxVerifyResult, 0),
		unProcessedTxs: make(map[common.Uint256]*ctypes.Transaction, 0),
	}
	return blk
}

// checkPendingBlockOk checks whether a block from consensus is verified.
// If some transaction is invalid, return the result directly at once, no
// need to wait for verifying the complete block.
func (self *BlockEntry) updateProcessedTx(hash common.Uint256, err errors.ErrCode) {

	tx, ok := self.unProcessedTxs[hash]
	if !ok {
		return
	}

	entry := &ttypes.TxVerifyResult{
		tx,
		ttypes.VerifyResult{self.height, err},
	}

	self.processedTxs[hash] = entry
	delete(self.unProcessedTxs, hash)

	// if the tx is invalid, send the response at once
	if err != errors.ErrNoError || len(self.unProcessedTxs) == 0 {
		self.send2Consensus()
	}
}

// verifyBlock verifies the block from consensus.
// There are three cases to handle.
// 1, for those unverified txs, assign them to the available worker;
// 2, for those verified txs whose height >= block's height, nothing to do;
// 3, for those verified txs whose height < block's height, re-verify their
// stateful data.
func (self *BlockEntry) updateBlock(verifiedResult []*ttypes.TxVerifyResult, unVerified, reVerify []*ctypes.Transaction, height uint32, txs []*ctypes.Transaction, consusActor *actor.PID) {

	self.consusActor = consusActor
	self.height = height
	self.processedTxs = make(map[common.Uint256]*ttypes.TxVerifyResult, len(txs))
	self.unProcessedTxs = make(map[common.Uint256]*ctypes.Transaction, 0)

	for _, t := range unVerified {
		//self.assignTxToWorker(t, ttypes.NilSender)
		self.unProcessedTxs[t.Hash()] = t
	}

	for _, t := range reVerify {
		//self.reVerifyStatefulTx(t, ttypes.NilSender)
		self.unProcessedTxs[t.Hash()] = t
	}

	for _, t := range verifiedResult {
		self.processedTxs[t.Tx.Hash()] = t
	}

	/* If all the txs in the blocks are verified, send response
	 * to the consensus directly
	 */
	if len(self.unProcessedTxs) == 0 {
		self.send2Consensus()
	}
}

// sendBlkResult2Consensus sends the result of verifying block to  consensus
func (self *BlockEntry) send2Consensus() {
	rsp := &ttypes.VerifyBlockRsp{
		TxResults: make([]*ttypes.TxVerifyResult, 0, len(self.processedTxs)),
	}
	for _, v := range self.processedTxs {
		rsp.TxResults = append(rsp.TxResults, v)
	}
	rsp0 := &txpoolcommon.VerifyBlockRsp{
		TxnPool: make([]*txpoolcommon.VerifyTxResult, 0, len(self.processedTxs)),
	}
	for _, v := range self.processedTxs {
		rsp0.TxnPool = append(rsp0.TxnPool, &txpoolcommon.VerifyTxResult{v.Height, v.Tx, v.ErrCode})
	}
	if self.consusActor != nil {
		self.consusActor.Tell(rsp0)
	}

	// Clear the processedTxs for the next block verify req
	for k := range self.processedTxs {
		delete(self.processedTxs, k)
	}
}
