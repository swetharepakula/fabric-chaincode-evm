Installing the EVMSCC with Fabric:

1. Use Fabric with `evmscc` from `https://github.com/swetharepakula/fabric/tree/master`. This is based off `fabric master`.
1. The  `sampleconfig/core.yml` being used to standup the network has already enabled `evmscc` (If using repo above).
```
    # system chaincodes whitelist. To add system chaincode "myscc" to the
    # whitelist, add "myscc: enable" to the list below, and register in
    # chaincode/importsysccs.go
    system:
        cscc: enable
        lscc: enable
        escc: enable
        vscc: enable
        qscc: enable
        evmscc: enable
```
1. Create a channel with the name `mychannel`. It is currently hardcoded as the channel to be used in the Fabric proxy


Interacting with Solidity Contracts in Fabric

Set up the Ethereum JSON RPC Server
1. Create Fabric SDK config to match the Fabric network. If running locally using `docker-compose`, you can do `docker ps -a` to get all the ip:port combinations you need. Here is a [sample]().
1. Set the environment variable ETHSERVER_CONFIG to the absolute path to the configuration you created in step 1.
1. Run the ethserver by `go run main.go`. This will start the server at `https://localhost:5000`. **NOTE** Go 1.9.4 and higher will result in a build error
**NOTE** The fabric proxy is hardcoded to use `mychannel` as the channel.

Set up your web3 console
1. Download and install web3 `npm install web3`
1. Connect to the ethserver created previously
  ```
  > Web3 = require('web3')
  > web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:5000"))
  ```
1. Pick a random 160 bit address and set the default account. The address does not matter because the account will be determined by the public key that was configured in the SDK config
  ```
  > web3.eth.defaultAccount = '0x8888f1f195afa192cfee860698584c030f4c9db1'
  ```

Install SimpleStorage Smart Contract:
1. `SimpleStorage` is the following contract:

```
pragma solidity ^0.4.0;

contract SimpleStorage {
    uint storedData;

    function set(uint x) public {
        storedData = x;
    }

    function get() public constant returns (uint) {
        return storedData;
    }
}
```
1. Create a contract object by using the abiDefinition, available at remix.ethereum.org.
```
> SimpleStorageABI = [
	{
		"constant": false,
		"inputs": [
			{
				"name": "x",
				"type": "uint256"
			}
		],
		"name": "set",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "get",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	}
]
> SimpleStorage = web3.eth.contract(SimpleStorageABI)
```
1. Using the ABI definition create a SimpleStorage contract
  ```
  > compiledBytecode = "6060604052341561000f57600080fd5b60d38061001d6000396000f3006060604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c14606e575b600080fd5b3415605857600080fd5b606c60048080359060200190919050506094565b005b3415607857600080fd5b607e609e565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a72305820e170013eadb8debdf58398ee9834aa86cf08db2eee5c90947c1bcf6c18e3eeff0029"
  > deployedContract = SimpleStorage.new([], {"data":compiledBytecode})
  > txReceipt = web3.eth.getTransactionReceipt(deployedContract.transactionHash)
  > deployedContract.address = txReceipt.ContractAddress
  > simpleStorage = SimpleStorage.at(deployedContract.address)
  ```

Interact with the smart contract by using the `simpleStorage` object
1. You can set a value:
```
simpleStorage.set(<insert number>)
```
1. You can get the value. In web3 everything is returned as a BigNumber so we do toNumber() to get it in a readible format.
```
simpleStorage.get().toNumber()
```

Interact with the smart contract through the peer cli:
1. Always use `peer chaincode invoke -c <yourchannel> `
1. To deploy a contract your args should be:
```
{"Args":["0000000000000000000000000000000000000000000000000000000000000000","<bytecode>"]}
```
The output of that is the contract address.
1. To interact with the contract your args should be:
```
{"Args":["<contract address>","<functionhash + arg>"]}
```
For SimpleStorage here are function hashes:
 - SET: `60fe47b1`  -> Takes one 32 byte argument Ex: To set the value as one, I would use: `60fe47b10000000000000000000000000000000000000000000000000000000000000001`
 - GET: `6d4ce63c` -> **NOTE** For GET, remember to add the --hex option so the output is readible.
