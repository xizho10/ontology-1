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

package actor

import (
	"errors"
	"time"

	"github.com/ontio/ontology-eventbus/actor"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/types"
	ontErrors "github.com/ontio/ontology/errors"
	ttypes "github.com/ontio/ontology/txnpool2/types"
)

//var txnPid *actor.PID
var txnPoolPid *actor.PID

//func SetTxPid(actr *actor.PID) {
//	txnPid = actr
//}
func SetTxnPoolPid(actr *actor.PID) {
	txnPoolPid = actr
}
func AppendTxToPool(txn *types.Transaction) ontErrors.ErrCode {
	txReq := &ttypes.AppendTxReq{
		Tx:         txn,
		HttpSender: true,
	}
	txnPoolPid.Tell(txReq)
	return ontErrors.ErrNoError
}

func GetTxsFromPool(byCount bool) (map[common.Uint256]*types.Transaction, uint64) {
	future := txnPoolPid.RequestFuture(&ttypes.GetVerifiedTxnFromPoolReq{ByCount: byCount}, REQ_TIMEOUT*time.Second)
	result, err := future.Result()
	if err != nil {
		log.Errorf(ERR_ACTOR_COMM, err)
		return nil, 0
	}
	txpool, ok := result.(*ttypes.GetVerifiedTxnFromPoolRsp)
	if !ok {
		return nil, 0
	}
	txMap := make(map[common.Uint256]*types.Transaction)
	var networkFeeSum uint64
	for _, v := range txpool.TxnPool {
		txMap[v.Tx.Hash()] = v.Tx
		networkFeeSum += v.Gas
	}
	return txMap, networkFeeSum

}

func GetTxFromPool(hash common.Uint256) (ttypes.TxEntry, error) {

	future := txnPoolPid.RequestFuture(&ttypes.GetVerifiedTxFromPoolReq{hash}, REQ_TIMEOUT*time.Second)
	result, err := future.Result()
	if err != nil {
		log.Errorf(ERR_ACTOR_COMM, err)
		return ttypes.TxEntry{}, err
	}
	rsp, ok := result.(*ttypes.GetVerifiedTxFromPoolRsp)
	if !ok {
		return ttypes.TxEntry{}, errors.New("fail")
	}
	if rsp.Txn == nil {
		return ttypes.TxEntry{}, errors.New("fail")
	}

	future = txnPoolPid.RequestFuture(&ttypes.GetTxVerifyResultReq{hash}, REQ_TIMEOUT*time.Second)
	result, err = future.Result()
	if err != nil {
		log.Errorf(ERR_ACTOR_COMM, err)
		return ttypes.TxEntry{}, err
	}
	txStatus, ok := result.(*ttypes.GetTxVerifyResultRsp)
	if !ok {
		return ttypes.TxEntry{}, errors.New("fail")
	}
	txnEntry := ttypes.TxEntry{Tx: rsp.Txn, VerifyHeight: txStatus.TxEntry.VerifyHeight}
	return txnEntry, nil
}

func GetTxCount() ([]uint64, error) {
	future := txnPoolPid.RequestFuture(&ttypes.GetTxVerifyResultStaticsReq{}, REQ_TIMEOUT*time.Second)
	result, err := future.Result()
	if err != nil {
		log.Errorf(ERR_ACTOR_COMM, err)
		return []uint64{}, err
	}
	txnCnt, ok := result.(*ttypes.GetTxVerifyResultStaticsRsp)
	if !ok {
		return []uint64{}, errors.New("fail")
	}
	return txnCnt.Count, nil
}
