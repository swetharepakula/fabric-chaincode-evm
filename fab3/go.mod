module github.com/hyperledger/fabric-chaincode-evm/fab3

replace github.com/perlin-network/life => github.com/silasdavis/life v0.0.0-20191009191257-e9c2a5fdbc96

replace github.com/hyperledger/fabric-chaincode-evm/evmcc => ../evmcc

require (
	github.com/cloudflare/cfssl v0.0.0-20180223231731-4e2dcbde5004 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/google/certificate-transparency-go v1.0.21 // indirect
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/rpc v1.2.0+incompatible
	github.com/hyperledger/burrow v0.29.1
	github.com/hyperledger/fabric v1.4.3
	github.com/hyperledger/fabric-chaincode-evm/evmcc v0.0.0
	github.com/hyperledger/fabric-sdk-go v1.0.0-alpha5
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.0
	github.com/pkg/errors v0.8.1
	github.com/zmap/zlint v1.0.2 // indirect
	go.uber.org/zap v1.10.0
)
