package actor

import (
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/events/message"
	tsend "github.com/ontio/ontology/txpool/actor/send"
	"github.com/ontio/ontology/txpool/pool"
	ttypes "github.com/ontio/ontology/txpool/types"
	vtypes "github.com/ontio/ontology/validator/types"
	"reflect"
)

// TxnPoolActor: Handle the high priority request from Consensus
type TxPoolActor struct {
	txPoolServer *pool.PoolServer
	sender       *tsend.Sender
}

// NewTxPoolActor creates an actor to handle the messages from the consensus
func NewTxPoolActor(svr *pool.PoolServer, sender *tsend.Sender) *TxPoolActor {
	t := &TxPoolActor{svr, sender}
	return t
}

// Receive implements the actor interface
func (self *TxPoolActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Started:
		log.Info("txpool actor started and be ready to receive txPool msg")

	case *actor.Stopping:
		log.Warn("txpool actor stopping")

	case *actor.Restarting:
		log.Warn("txpool actor Restarting")

	case *ttypes.GetVerifiedTxnFromPoolReq:
		sender := context.Sender()
		log.Debug("txpool actor Receives getting tx pool req from ", sender)
		res := self.txPoolServer.HandleGetTxEntrysFromPool(msg.ByCount, msg.Height)
		if sender != nil {
			sender.Request(&ttypes.GetVerifiedTxnFromPoolRsp{TxnPool: res}, context.Self())
		}

	case *ttypes.VerifyBlockReq:
		sender := context.Sender()
		log.Debug("txpool actor Receives verifying block req from ", sender)
		if msg == nil || len(msg.Txs) == 0 {
			return
		}
		self.txPoolServer.HandleVerifyBlockReq(msg.Height, msg.Txs, sender)

	case *message.SaveBlockCompleteMsg:
		sender := context.Sender()
		log.Debug("txpool actor Receives block complete event from ", sender)
		if msg.Block != nil {
			self.txPoolServer.HandleSaveBlockComplete(msg.Block.Transactions)
		}
		//below is tx status
	case *ttypes.AppendTxReq:
		sender := msg.HttpSender
		log.Debug("txpool-tx actor Receives tx from ", sender)
		self.txPoolServer.HandleAppendTxReq(sender, msg.Tx)

	case *ttypes.GetVerifiedTxFromPoolReq:
		sender := context.Sender()
		log.Debug("txpool-tx actor Receives getting tx req from ", sender)
		res := self.txPoolServer.HandleGetVerifiedTxFromPool(msg.Hash)
		if sender != nil {
			sender.Request(&ttypes.GetVerifiedTxFromPoolRsp{Txn: res},
				context.Self())
		}

	case *ttypes.GetTxVerifyResultStaticsReq:
		sender := context.Sender()
		log.Debug("txpool-tx actor Receives getting tx stats from ", sender)
		res := self.txPoolServer.HandleGetStatistics()
		if sender != nil {
			sender.Request(&ttypes.GetTxVerifyResultStaticsRsp{Count: res},
				context.Self())
		}

	case *ttypes.IsTxInPoolReq:
		sender := context.Sender()
		log.Debug("txpool-tx actor Receives checking tx req from ", sender)
		res := self.txPoolServer.HandleIsContainTx(msg.Hash)
		if sender != nil {
			sender.Request(&ttypes.IsTxInPoolRsp{Ok: res},
				context.Self())
		}

	case *ttypes.GetTxVerifyResultReq:
		sender := context.Sender()
		log.Debug("txpool-tx actor Receives getting tx status req from ", sender)
		txEntry := self.txPoolServer.HandleGetTxVerifyResult(msg.Hash)
		if sender != nil {
			sender.Request(&ttypes.GetTxVerifyResultRsp{Hash: msg.Hash,
				TxEntry: txEntry,
			}, context.Self())
		}

	case *vtypes.RegisterValidatorReq:
		log.Debugf("txpool-verify actor:: validator %v connected", msg.Validator)
		self.sender.HandleRegisterValidator(msg.Validator, msg.Type, msg.Id)

	case *vtypes.UnRegisterValidatorReq:
		log.Debugf("txpool-verify actor:: validator %d:%v disconnected", msg.VerifyType, msg.Id)

		self.sender.HandleUnRegisterValidator(msg.VerifyType, msg.Id)

	case *vtypes.VerifyTxRsp:
		log.Debug("txpool-verify actor:: Receives verify rsp message")

		self.txPoolServer.HandleVerifyTxRsp(msg)

	default:
		log.Debug("txpool actor: Unknown msg ", msg, "type", reflect.TypeOf(msg))
	}
	//update pool
	self.txPoolServer.HandleUpdatePool()
}

func (self *TxPoolActor) setServer(svr *pool.PoolServer) {
	self.txPoolServer = svr
}
