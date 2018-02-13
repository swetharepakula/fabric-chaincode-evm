package ethserver

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
)

type EthRPCService struct {
	EthService
}

type EthService interface {
	GetCode(*http.Request, *DataParam, *string) error
	Call(*http.Request, *Params, *string) error
	SendTransaction(*http.Request, *Params, *string) error
	GetTransactionReceipt(*http.Request, *DataParam, *TxReceipt) error
}

type ethRPCService struct {
	sdk *fabsdk.FabricSDK
}

type DataParam string
type Params struct {
	From     string
	To       string
	Gas      string
	GasPrice string
	Value    string
	Data     string
	Nonce    string
}

type TxReceipt struct {
	TransactionHash string
	BlockHash       string
	BlockNumber     string
	ContractAddress string
}

type EthServer struct {
	Server   *rpc.Server
	listener net.Listener
}

var defaultUser = "User1"
var channelID = "mychannel"

func NewEthService(configFile string) EthService {
	fmt.Println(configFile)
	c := config.FromFile(configFile)
	sdk, err := fabsdk.New(c)
	if err != nil {
		log.Panic("error creating sdk: ", err)
	}

	return &ethRPCService{
		sdk: sdk,
	}
}

func NewEthServer(eth EthService) *EthServer {
	server := rpc.NewServer()

	ethService := EthRPCService{eth}
	server.RegisterCodec(NewRPCCodec(), "application/json")
	server.RegisterService(ethService, "eth")

	return &EthServer{
		Server: server,
	}
}

func (s *EthServer) Start(port int) {
	r := mux.NewRouter()
	r.Handle("/", s.Server)

	http.ListenAndServe(fmt.Sprintf(":%d", port), r)
}

func (req *ethRPCService) GetCode(r *http.Request, args *DataParam, reply *string) error {

	chClient, err := req.sdk.NewChannelClient(channelID, defaultUser)
	if err != nil {
		log.Panic("error creating client", err)
	}

	defer chClient.Close()

	queryArgs := [][]byte{[]byte(*args)}

	value, err := Query(chClient, "evmscc", "getCode", queryArgs)
	if err != nil {
		fmt.Printf("Failed to query: %s\n", err)
	}
	*reply = string(value)

	return nil
}

func (req *ethRPCService) Call(r *http.Request, params *Params, reply *string) error {

	chClient, err := req.sdk.NewChannelClient(channelID, defaultUser)
	if err != nil {
		return err
	}
	defer chClient.Close()

	args := [][]byte{[]byte(params.Data)}

	value, err := Query(chClient, "evmscc", params.To, args)
	if err != nil {
		return err
	}

	*reply = string(value)

	return nil
}

func (req *ethRPCService) SendTransaction(r *http.Request, params *Params, reply *string) error {

	chClient, err := req.sdk.NewChannelClient(channelID, defaultUser)
	if err != nil {
		return err
	}
	defer chClient.Close()

	txReq := apitxn.ExecuteTxRequest{
		ChaincodeID: "evmscc",
		Fcn:         params.To,
		Args:        [][]byte{[]byte(params.Data)},
	}

	//Return only the transaction ID
	//Maybe change to an async transaction
	_, txID, err := chClient.ExecuteTx(txReq)
	if err != nil {
		return err
	}

	*reply = txID.ID

	return nil
}

//TODO: Return only the transaction result in the Contract Address spot.
func (req *ethRPCService) GetTransactionReceipt(r *http.Request, param *DataParam, reply *TxReceipt) error {

	chClient, err := req.sdk.NewChannelClient(channelID, defaultUser)

	args := [][]byte{[]byte(channelID), []byte(*param)}

	t, err := Query(chClient, "qscc", "GetTransactionByID", args)
	if err != nil {
		return err
	}

	tx := &peer.ProcessedTransaction{}
	err = proto.Unmarshal(t, tx)
	if err != nil {
		return err
	}

	b, err := Query(chClient, "qscc", "GetBlockByTxID", args)
	if err != nil {
		return err
	}

	block := &common.Block{}
	err = proto.Unmarshal(b, block)
	if err != nil {
		return err
	}

	blkHeader := block.GetHeader()

	p := tx.GetTransactionEnvelope().GetPayload()
	payload := &common.Payload{}
	err = proto.Unmarshal(p, payload)
	if err != nil {
		return err
	}

	fmt.Printf("PAYLOAD HEADERS: %+v\n\n", payload.GetHeader())

	// have to figure out when to pass in the contract address or not
	*reply = TxReceipt{
		TransactionHash: string(*param),
		BlockHash:       hex.EncodeToString(blkHeader.Hash()),
		BlockNumber:     strconv.FormatUint(blkHeader.GetNumber(), 10),
		ContractAddress: string(payload.GetData()),
	}

	return nil

}

func Query(chClient apitxn.ChannelClient, chaincodeID string, function string, queryArgs [][]byte) ([]byte, error) {

	return chClient.Query(apitxn.QueryRequest{
		ChaincodeID: chaincodeID,
		Fcn:         function,
		Args:        queryArgs,
	})
}
