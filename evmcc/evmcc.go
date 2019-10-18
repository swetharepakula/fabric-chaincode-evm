/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/execution/engine"
	"github.com/hyperledger/burrow/execution/evm"
	"github.com/hyperledger/burrow/genesis"
	"github.com/hyperledger/burrow/logging"
	"github.com/hyperledger/burrow/permission"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/hyperledger/fabric-chaincode-evm/evmcc/address"
	"github.com/hyperledger/fabric-chaincode-evm/evmcc/eventmanager"
	"github.com/hyperledger/fabric-chaincode-evm/evmcc/statemanager"
)

//Permissions for all accounts (users & contracts) to send CallTx or SendTx to a contract
const ContractPermFlags = permission.Call | permission.Send | permission.CreateContract

var ContractPerms = permission.AccountPermissions{
	Base: permission.BasePermissions{
		Perms:  ContractPermFlags,
		SetBit: ContractPermFlags,
	},
}

var logger = flogging.MustGetLogger("evmcc")
var evmLogger = logging.NewNoopLogger()

// Blockchain interface required to run EVM
// Currently not supported as current BlockHeight and BlockTime can lead to undeterministic output
type blockchain struct{}

func (*blockchain) LastBlockHeight() uint64 {
	panic("Block Height shouldn't be called")
}

func (*blockchain) LastBlockTime() time.Time {
	panic("Block Time shouldn't be called")
}

func (*blockchain) BlockHash(height uint64) ([]byte, error) {
	panic("Block Hash shouldn't be called")
}

type EvmChaincode struct{}

func (evmcc *EvmChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	defaultPermissionsAccount := genesis.PermissionsAccount(ContractPerms)
	encodedAcct, err := defaultPermissionsAccount.Marshal()
	if err != nil {
		shim.Error(fmt.Sprintf("failed to marshal default permissions account: %s", err))
	}

	stub.PutState(hex.EncodeToString(defaultPermissionsAccount.Address.Bytes()), encodedAcct)

	logger.Debugf("Init evmcc, it's no-op")
	return shim.Success(nil)
}

func (evmcc *EvmChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	// We always expect 2 args: 'callee address, input data' or ' getCode ,  contract address'
	args := stub.GetArgs()

	if len(args) == 1 {
		if string(args[0]) == "account" {
			return evmcc.account(stub)
		}
	}

	if len(args) != 2 {
		return shim.Error(fmt.Sprintf("expects 2 args, got %d : %s", len(args), string(args[0])))
	}

	if string(args[0]) == "getCode" {
		return evmcc.getCode(stub, args[1])
	}

	c, err := hex.DecodeString(string(args[0]))
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to decode callee address from %s: %s", string(args[0]), err))
	}

	calleeAddr, err := crypto.AddressFromBytes(c)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get callee address: %s", err))
	}

	// get caller account from creator public key
	callerAddr, err := getCallerAddress(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get caller address: %s", err))
	}

	// get input bytes from args[1]
	input, err := hex.DecodeString(string(args[1]))
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to decode input bytes: %s", err))
	}

	var gas uint64 = 10000
	state := statemanager.NewStateManager(stub)
	eventSink := &eventmanager.EventManager{Stub: stub}
	nonce := crypto.Nonce(callerAddr, []byte(stub.GetTxID()))
	// vm := evm.NewVM(newParams(), callerAddr, nonce, evmLogger)
	vm := evm.New(evm.Options{})

	if calleeAddr == crypto.ZeroAddress {
		logger.Debugf("Deploy contract")

		logger.Debugf("Contract nonce number = %d", nonce)
		contractAddr := crypto.NewContractAddress(callerAddr, nonce)
		// Contract account needs to be created before setting code to it
		perms := permission.NewAccountPermissions(ContractPermFlags)
		acc := &acm.Account{Address: contractAddr, Permissions: perms}
		err := state.UpdateAccount(acc)
		if err != nil {
			return shim.Error(fmt.Sprintf("failed to create the contract account: %s ", err))
		}

		callParams := engine.CallParams{
			Origin: callerAddr,
			Caller: callerAddr,
			Callee: contractAddr,
			Input:  input,
			Gas:    &gas,
		}

		rtCode, evmErr := vm.Execute(state, &blockchain{}, eventSink, callParams, input)
		if evmErr != nil {
			return shim.Error(fmt.Sprintf("failed to deploy code: %s", evmErr))
		}
		if rtCode == nil {
			return shim.Error(fmt.Sprintf("nil bytecode"))
		}

		acc.EVMCode = rtCode
		err = state.UpdateAccount(acc)
		if err != nil {
			return shim.Error(fmt.Sprintf("failed to update the contract account with code: %s ", err))
		}

		// Passing the first 4 bytes contract address just created
		// Since the bytes are not hex encoded, one byte will be represented
		// as 2 hex bytes, so the event name will be 8 hex bytes.
		// Hex Encode before flushing to ensure no non utf-8 characters
		// Otherwise proto marshal fails on non utf-8 characters when
		// the peer tries to marshal the event
		err = eventSink.Flush(hex.EncodeToString(contractAddr.Bytes()[0:4]))
		if err != nil {
			return shim.Error(fmt.Sprintf("error in Flush: %s", err))
		}

		// return encoded hex bytes for human-readability
		return shim.Success([]byte(hex.EncodeToString(contractAddr.Bytes())))
	} else {
		logger.Debugf("Invoke contract at %x", calleeAddr.Bytes())

		calleeAcct, err := state.GetAccount(calleeAddr)
		if err != nil {
			return shim.Error(fmt.Sprintf("failed to retrieve contract code: %s", err))
		}

		callParams := engine.CallParams{
			Origin: callerAddr,
			Caller: callerAddr,
			Callee: calleeAddr,
			Input:  input,
			Gas:    &gas,
		}

		output, err := vm.Execute(state, &blockchain{}, eventSink, callParams, calleeAcct.EVMCode)
		if err != nil {
			return shim.Error(fmt.Sprintf("failed to execute contract: %s", err))
		}

		// Passing the function hash of the method that has triggered the event
		// The function hash is the first 8 bytes of the Input argument
		// The argument is a hex-encoded evm function hash, so we can directly pass the bytes
		err = eventSink.Flush(string(args[1][0:8]))
		if err != nil {
			return shim.Error(fmt.Sprintf("error in Flush: %s", err))
		}

		return shim.Success(output)
	}
}

func (evmcc *EvmChaincode) getCode(stub shim.ChaincodeStubInterface, address []byte) pb.Response {
	c, err := hex.DecodeString(string(address))
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to decode callee address from %s: %s", string(address), err))
	}

	calleeAddr, err := crypto.AddressFromBytes(c)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get callee address: %s", err))
	}

	acctBytes, err := stub.GetState(strings.ToLower(calleeAddr.String()))
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get contract account: %s", err))
	}

	if len(acctBytes) == 0 {
		return shim.Success(acctBytes)
	}

	acct := &acm.Account{}
	err = acct.Unmarshal(acctBytes)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to decode contract account: %s", err))
	}

	return shim.Success([]byte(hex.EncodeToString(acct.EVMCode.Bytes())))
}

func (evmcc *EvmChaincode) account(stub shim.ChaincodeStubInterface) pb.Response {
	callerAddr, err := getCallerAddress(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("fail to convert identity to address: %s", err))
	}
	return shim.Success([]byte(callerAddr.String()))
}

func getCallerAddress(stub shim.ChaincodeStubInterface) (crypto.Address, error) {
	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return crypto.ZeroAddress, fmt.Errorf("failed to get creator: %s", err)
	}

	callerAddr, err := address.IdentityToAddr(creatorBytes)
	if err != nil {
		return crypto.ZeroAddress, fmt.Errorf("fail to convert identity to address: %s", err)
	}

	return crypto.AddressFromBytes(callerAddr)
}

func main() {
	if err := shim.Start(new(EvmChaincode)); err != nil {
		logger.Infof("Error starting EVM chaincode: %s", err)
	}
}
