module github.com/hyperledger/fabric-chaincode-evm/fab3

replace github.com/perlin-network/life => github.com/silasdavis/life v0.0.0-20191009191257-e9c2a5fdbc96

replace github.com/hyperledger/fabric-chaincode-evm/evmcc => ../evmcc

require (
	github.com/cloudflare/cfssl v0.0.0-20180223231731-4e2dcbde5004 // indirect from fabric-go-sdk
	github.com/gogo/protobuf v1.3.1
	github.com/google/certificate-transparency-go v1.0.21 // indirect
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/rpc v1.2.0+incompatible
	github.com/hyperledger/burrow v0.29.1
	github.com/hyperledger/fabric v1.4.3
	github.com/hyperledger/fabric-chaincode-evm/evmcc v0.0.0
	github.com/hyperledger/fabric-lib-go v1.0.0 // indirect
	github.com/hyperledger/fabric-sdk-go v1.0.0-alpha5
	github.com/ianlancetaylor/demangle v0.0.0-20181102032728-5e5cf60278f6 // indirect
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.2.0 //indirect from fabric go sdk
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/zmap/zlint v1.0.2 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/arch v0.0.0-20190927153633-4e8777c89be4 // indirect
)
