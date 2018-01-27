package main

import (
	"os"

	"github.com/hyperledger/fabric-chaincode-evm/ethserver"
)

func main() {
	configFile := os.Getenv("ETHSERVER_CONFIG")
	ethService := ethserver.NewEthService(configFile)
	server := ethserver.NewEthServer(ethService)

	server.Start(5000)
}
