module github.com/hyperledger/fabric-chaincode-evm/integration

require (
	code.cloudfoundry.org/clock v0.0.0-20180518195852-02e53af36e6c // indirect
	github.com/containerd/continuity v0.0.0-20190827140505-75bee3e2ccb6 // indirect
	github.com/fsouza/go-dockerclient v1.4.4
	github.com/hyperledger/fabric v1.4.0
	github.com/hyperledger/fabric-chaincode-evm/fab3 v0.0.0
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.0
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00
	github.com/vishvananda/netns v0.0.0-20190625233234-7109fa855b0f // indirect
	go.etcd.io/etcd v3.3.17+incompatible // indirect
	golang.org/x/tools v0.0.0-20191017205301-920acffc3e65 // indirect
)

replace github.com/hyperledger/fabric-chaincode-evm/fab3 => ../fab3

replace github.com/hyperledger/fabric-chaincode-evm/evmcc => ../evmcc

replace github.com/perlin-network/life => github.com/silasdavis/life v0.0.0-20191009191257-e9c2a5fdbc96

replace github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.8.0

replace github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
