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

import ()

// ActorType enumerates the kind of actor
type ActorType uint8

const (
	_              ActorType = iota
	TxActor                  // Actor that handles new transaction
	TxPoolActor              // Actor that handles consensus msg
	VerifyRspActor           // Actor that handles the response from valdiators
	NetActor                 // Actor to send msg to the net actor
	MaxActor
)
