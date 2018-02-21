Interacting with Solidity Contracts in Fabric

Set up the Ethereum JSON RPC Server
1. Create Fabric SDK config to match the Fabric network. If running locally using `docker-compose`, you can do `docker ps -a` to get all the ip:port combinations you need.
1. Set the environment variable ETHSERVER_CONFIG to the absolute path to the configuration you created in step 1.
1. Run the ethserver by `go run main.go`. This will start the server at `https://localhost:5000`.

Set up your web3 console
1. Download and install web3 `npm install web3`
1. Connect to the ethserver created previously
  ```
  > Web3 = require('web3')
  > web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:5000"))
  ```
1. Pick a random 160 bit address and set the default account. The address does not matter because the account will be deteremined by the public key that was configured in the SDK config
  ```
  > web3.eth.defaultAccount = '0x8888f1f195afa192cfee860698584c030f4c9db1'
  ```

Install a smart contract:
1. Create a contract object by using the abiDefinition, available at remix.ethereum.org
  ```
  > SampleContract = web3.eth.contract(abiDefinition)
  > deployedContract = SampleContract.new([<params for init>], {"data":"<compiled contract bytecode>"})
  > txReceipt = web3.eth.getTransactionReceipt(deployedContract.transactionHash)
  > deployedContract.address = txReceipt.ContractAddress
  > contractInstance = SampleContract.at(deployedContract.address)
  ```

Interact with the smart contract by using the `SampleContract` object
