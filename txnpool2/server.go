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

// Package txnpool privides a function to start micro service txPool for
// external process
package txnpool2

import (
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/events"
	"github.com/ontio/ontology/events/message"
	tactor "github.com/ontio/ontology/txnpool2/actor"
	tc "github.com/ontio/ontology/txnpool2/common"
	"github.com/ontio/ontology/txnpool2/pool"
	tsend "github.com/ontio/ontology/txnpool2/proc"
	"goproject/src/github.com/kataras/go-errors"
)

// startActor starts an actor with the proxy and unique id,
// and return the pid.
func startActor(obj interface{}, id string) *actor.PID {
	props := actor.FromProducer(func() actor.Actor {
		return obj.(actor.Actor)
	})

	pid, _ := actor.SpawnNamed(props, id)
	if pid == nil {
		log.Error("Fail to start actor")
		return nil
	}
	return pid
}

// StartTxnPoolServer starts the txnpool server and registers
// actors to handle the msgs from the network, http, consensus
// and validators. Meanwhile subscribes the block complete  event.
func StartTxnPoolServer() (*tsend.TXPoolServer, error) {
	sender := tsend.NewSender()
	/* Start txnpool server to receive msgs from p2p,
	 * consensus and valdiators
	 */
	server := pool.NewTxPoolServer(sender)

	// Initialize an actor to handle the msgs from consensus
	tpa := tactor.NewTxPoolActor(server, sender)
	txPoolPid := startActor(tpa, "txPool")
	if txPoolPid == nil {
		log.Error("Fail to start txnpool actor")
		return nil, errors.New("fail")
	}
	sender.RegisterActor(tc.TxPoolActor, txPoolPid)

	// Subscribe the block complete event
	var sub = events.NewActorSubscriber(txPoolPid)
	sub.Subscribe(message.TOPIC_SAVE_BLOCK_COMPLETE)
	return sender, nil
}
