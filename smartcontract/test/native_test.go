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

package test

import (
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/smartcontract"
	//"github.com/stretchr/testify/assert"
	"fmt"
	"github.com/ontio/ontology/vm/neovm/types"
	"os"
	"testing"
)

func TestBuildParamToNative(t *testing.T) {
	code := `00c57676c84c0500000000004c1400000000000000000000000000000000000000060068164f6e746f6c6f67792e4e61746976652e496e766f6b65`
	code = `58c56b6a00527ac46a51527ac46a00c30548656c6c6f9c6416006a51c300c36a52527ac46a52c3650b006c756661006c756655c56b6a00527ac46a00c3681553797374656d2e52756e74696d652e4e6f7469667961516c7566`

	hex, err := common.HexToBytes(code)

	if err != nil {
		t.Fatal("hex to byte error:", err)
	}

	config := &smartcontract.Config{
		Time:   10,
		Height: 10,
		Tx:     nil,
	}
	//cache := storage.NewCloneCache(testBatch)
	sc := smartcontract.SmartContract{
		Config: config,
		Gas:    100000,
	}

	engine, err := sc.NewExecuteEngine(hex)

	result, err := engine.Invoke()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	fmt.Println(result.(types.StackItems).GetBigInteger())
	//	assert.Error(t, err, "invoke smart contract err: [NeoVmService] service system call error!: [SystemCall] service execute error!: invoke native circular reference!")
}
