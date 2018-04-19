package main

import (
	"os"

	"github.com/hyperledger/fabric-chaincode-evm/ethserver"
)

func main() {
	configFile := os.Getenv("ETHSERVER_CONFIG")
	user := os.Getenv("ETHSERVER_USER")
	if user == "" {
		user = "User1"
	}
	ethService := ethserver.NewEthService(configFile, user)
	server := ethserver.NewEthServer(ethService)

	server.Start(5000)
}
