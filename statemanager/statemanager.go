/*
Copyright IBM Corp. 2016 All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
		 http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package statemanager

import (
	"github.com/hyperledger/burrow/account"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type StateManager interface {
	GetAccount(address account.Address) (account.Account, error)
	GetStorage(address account.Address, key binary.Word256) (binary.Word256, error)
	UpdateAccount(updatedAccount account.Account) error
	RemoveAccount(address account.Address) error
	SetStorage(address account.Address, key, value binary.Word256) error
}

type stateManager struct {
	stub shim.ChaincodeStubInterface
}

func NewStateManager(stub shim.ChaincodeStubInterface) StateManager {
	return &stateManager{stub: stub}
}

func (s *stateManager) GetAccount(address account.Address) (account.Account, error) {
	code, err := s.stub.GetState(address.String())
	if err != nil {
		return account.ConcreteAccount{}.Account(), err
	}

	if len(code) == 0 {
		return account.ConcreteAccount{}.Account(), nil
	}

	return account.ConcreteAccount{
		Address: address,
		Code:    code,
	}.Account(), nil
}

func (s *stateManager) GetStorage(address account.Address, key binary.Word256) (binary.Word256, error) {
	compKey := address.String() + key.String()

	val, err := s.stub.GetState(compKey)
	if err != nil {
		return binary.Word256{}, err
	}
	return binary.LeftPadWord256(val), nil
}

func (s *stateManager) UpdateAccount(updatedAccount account.Account) error {

	return s.stub.PutState(updatedAccount.Address().String(), updatedAccount.Code().Bytes())
}

func (s *stateManager) RemoveAccount(address account.Address) error {
	return s.stub.DelState(address.String())
}

func (s *stateManager) SetStorage(address account.Address, key, value binary.Word256) error {
	compKey := address.String() + key.String()
	return s.stub.PutState(compKey, value.Bytes())
}
