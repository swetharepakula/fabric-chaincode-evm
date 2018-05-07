/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"

	"github.com/hyperledger/fabric-chaincode-evm/ethserver"
)

func main() {
	configFile := os.Getenv("ETHSERVER_CONFIG")
	user := os.Getenv("ETHSERVER_USER")
	if user == "" {
		user = "9ab9dd6465daf96f9c53abd1d21f5cd2bc0df4ee"
	}
	ethService := ethserver.NewEthService(configFile, user)
	server := ethserver.NewEthServer(ethService)

	server.Start(5000)
}
