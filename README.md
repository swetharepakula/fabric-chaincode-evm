## Hyperledger Fabric Shim for node.js chaincodes

This is the project for the fabric chaincode shim for the Burrow EVM.

Please see the draft and evolving design document in [FAB-6590](https://jira.hyperledger.org/browse/FAB-6590).

## Running the Proxy:
In the root directory of this repo (fabric-chaincode-evm) run:
```
ETHSERVER_CONFIG=<path to cluster sdk config> go run main.go
```
**NOTE** You need a GO Version that is less 1.9.4.

### Optional Environment Variables:
```
PORT              -- Proxy will run on the port specified on the environment variable. Default is 5000.
ETHSERVER_USER    -- Proxy will use the user id specfied on the environment variable. The user id corresponds to the name of the directories under the crypto-config/peerOrganizations/org1.example.com/users/Default is USER1.
ETHSERVER_CHANNEL -- Proxy will use the channel specified on the environment variable. Default is channel1
```

## Instructions to Run the Sample Voting App:

**NOTE** You need the node.js library `web3` version 0.20.2 installed.

### Set Environment Vairables
Use environment variables to choose what fabproxy to for the app to contact.
```
ETHSERVER_USER=User1 # App will use http://localhost:5000 as its provider
ETHSERVER_USER=User2 # App will use http://localhost:5001 as its provider
```
### Using the App
```
node voter/app.js <command> <args>
  Available Commands:
    deploy                                            # Deploys the Voting Contract
    giveRightToVote <contract-address> <user-address> # Allow <user-address> to vote
    vote <contract-address> <proposal-number>         # Vote for <proposal-number>
```


<a rel="license" href="http://creativecommons.org/licenses/by/4.0/"><img alt="Creative Commons License" style="border-width:0" src="https://i.creativecommons.org/l/by/4.0/88x31.png" /></a><br />This work is licensed under a <a rel="license" href="http://creativecommons.org/licenses/by/4.0/">Creative Commons Attribution 4.0 International License</a>
