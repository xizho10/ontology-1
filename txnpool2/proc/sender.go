package proc

import (
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/ontio/ontology/common/log"
	ctypes "github.com/ontio/ontology/core/types"
	//ttypes "github.com/ontio/ontology/txnpool2/types"
	tc "github.com/ontio/ontology/txnpool2/common"
	vtypes "github.com/ontio/ontology/validator2/types"
	"sync"
)

type validator struct {
	Validator *actor.PID
	Type      vtypes.VerifyType
	Id        string
}
type validators struct {
	sync.RWMutex
	entries    map[vtypes.VerifyType][]*validator // Registered validator container
	robinState map[vtypes.VerifyType]int          // Keep the round robin index for each verify type
}

type TXPoolServer struct {
	// The registered validators
	validatorActor *validators
	txPoolActor    *actor.PID
	//verifyRspActor *actor.PID
	netActor *actor.PID
}

func NewSender() *TXPoolServer {
	send := &TXPoolServer{
		validatorActor: &validators{
			entries:    make(map[vtypes.VerifyType][]*validator),
			robinState: make(map[vtypes.VerifyType]int),
		},
	}
	return send
}

// RegisterActor registers an actor with the actor type and pid.
func (self *TXPoolServer) RegisterActor(tpe tc.ActorType, pid *actor.PID) {
	if tpe == tc.TxActor {
		self.txPoolActor = pid
	} else if tpe == tc.TxPoolActor {
		self.txPoolActor = pid
	} else if tpe == tc.VerifyRspActor {
		self.txPoolActor = pid
	} else if tpe == tc.NetActor {
		self.netActor = pid
	}
}

// UnRegisterActor cancels the actor with the actor type.
func (self *TXPoolServer) UnRegisterActor(tpe tc.ActorType) {
	if tpe == tc.TxActor {
		self.txPoolActor = nil
	} else if tpe == tc.TxPoolActor {
		self.txPoolActor = nil
	} else if tpe == tc.VerifyRspActor {
		self.txPoolActor = nil
	} else if tpe == tc.NetActor {
		self.netActor = nil
	}
}

func (self *TXPoolServer) SendTxToNetActor(tx *ctypes.Transaction) {
	if self.netActor != nil {
		self.netActor.Tell(tx)
	}
}

// GetPID returns an actor pid with the actor type, If the type
// doesn't exist, return nil.
func (self *TXPoolServer) GetPID(tpe tc.ActorType) *actor.PID {
	if tpe == tc.TxPoolActor {
		return self.txPoolActor
	} else if tpe == tc.TxActor {
		return self.txPoolActor
	} else if tpe == tc.VerifyRspActor {
		return self.txPoolActor
	} else if tpe == tc.NetActor {
		return self.netActor
	}
	return nil
}
func (self *TXPoolServer) SendVerifyTxReq(verifyType vtypes.VerifyType, tx *ctypes.Transaction) bool {
	if verifyType == vtypes.Stateful {
		return self.sendVerifyStatefulTxReq(tx)
	} else if verifyType == vtypes.Stateless {
		return self.sendVerifyStatelessTxReq(tx)
	}
	return false
}

// sendReq2Validator sends a check request to the validators
func (self *TXPoolServer) sendVerifyStatelessTxReq(tx *ctypes.Transaction) bool {
	req := &vtypes.VerifyTxReq{
		Tx: *tx,
	}
	rspPid := self.txPoolActor
	if rspPid == nil {
		log.Error("VerifyRspActor not exist")
		return false
	}

	pids := self.getNextValidator()
	if pids == nil {
		log.Error("VerifyRspActor not exist")
		return false
	}
	for _, pid := range pids {
		pid.Request(req, rspPid)
	}

	return true
}

// sendReq2StatefulV sends a check request to the stateful validator
func (self *TXPoolServer) sendVerifyStatefulTxReq(tx *ctypes.Transaction) bool {
	req := &vtypes.VerifyTxReq{
		Tx: *tx,
	}
	rspPid := self.txPoolActor
	if rspPid == nil {
		log.Info("VerifyRspActor not exist")
		return false
	}

	pid := self.getNextValidatorByType(vtypes.Stateful)
	log.Info("send tx to the stateful")
	if pid == nil {
		return false
	}

	pid.Request(req, rspPid)
	return true
}

// getNextValidatorPIDs returns the next pids to verify the transaction using
// roundRobin LB.
//return two type stateful and stateless validoter
func (self *TXPoolServer) getNextValidator() []*actor.PID {
	self.validatorActor.Lock()
	defer self.validatorActor.Unlock()

	if len(self.validatorActor.entries) == 0 {
		return nil
	}

	pids := make([]*actor.PID, 0, len(self.validatorActor.entries))
	for k, v := range self.validatorActor.entries {
		preIndex := self.validatorActor.robinState[k]
		nextIndex := (preIndex + 1) % len(v)
		self.validatorActor.robinState[k] = nextIndex
		pids = append(pids, v[nextIndex].Validator)
	}
	return pids
}

// getNextValidatorPID returns the next pid with the verify type using roundRobin LB
func (self *TXPoolServer) getNextValidatorByType(key vtypes.VerifyType) *actor.PID {
	self.validatorActor.Lock()
	defer self.validatorActor.Unlock()

	length := len(self.validatorActor.entries[key])
	if length == 0 {
		return nil
	}

	entries := self.validatorActor.entries[key]
	preIndex := self.validatorActor.robinState[key]
	nextIndex := (preIndex + 1) % length
	self.validatorActor.robinState[key] = nextIndex
	return entries[nextIndex].Validator
}

// registerValidator registers a validator to verify a transaction.
func (self *TXPoolServer) HandleRegisterValidator(pid *actor.PID, tpe vtypes.VerifyType, id string) {
	self.validatorActor.Lock()
	defer self.validatorActor.Unlock()

	_, ok := self.validatorActor.entries[tpe]

	if !ok {
		self.validatorActor.entries[tpe] = make([]*validator, 0, 1)
	}
	self.validatorActor.entries[tpe] = append(self.validatorActor.entries[tpe], &validator{pid, tpe, id})
}

// unRegisterValidator cancels a validator with the verify type and id.
func (self *TXPoolServer) HandleUnRegisterValidator(verifyType vtypes.VerifyType,
	id string) {

	self.validatorActor.Lock()
	defer self.validatorActor.Unlock()

	tmpSlice, ok := self.validatorActor.entries[verifyType]
	if !ok {
		log.Error("No validator on check type:%d\n", verifyType)
		return
	}

	for i, v := range tmpSlice {
		if v.Id == id {
			self.validatorActor.entries[verifyType] =
				append(tmpSlice[0:i], tmpSlice[i+1:]...)
			if v.Validator != nil {
				v.Validator.Tell(&vtypes.UnRegisterValidatorRsp{Id: id, VerifyType: verifyType})
			}
			if len(self.validatorActor.entries[verifyType]) == 0 {
				delete(self.validatorActor.entries, verifyType)
			}
		}
	}
}

func (self *TXPoolServer) Stop() {
	self.netActor.Stop()
	self.txPoolActor.Stop()
	self.txPoolActor.Stop()
	self.netActor.Stop()
}
