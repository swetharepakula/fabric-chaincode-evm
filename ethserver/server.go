package ethserver

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
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
var zeroAddress = make([]byte, 20)

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

	queryArgs := [][]byte{[]byte(Strip0xFromHex(string(*args)))}

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

	args := [][]byte{[]byte(Strip0xFromHex(params.Data))}

	value, err := Query(chClient, "evmscc", Strip0xFromHex(params.To), args)
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

	if params.To == "" {
		params.To = hex.EncodeToString(zeroAddress)
	}

	txReq := apitxn.ExecuteTxRequest{
		ChaincodeID: "evmscc",
		Fcn:         Strip0xFromHex(params.To),
		Args:        [][]byte{[]byte(Strip0xFromHex(params.Data))},
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

	txActions := &peer.Transaction{}
	err = proto.Unmarshal(payload.GetData(), txActions)

	if err != nil {
		return err
	}

	actions := txActions.GetActions()

	ccPropPayload, respPayload, err := GetPayloads(actions[0])
	if err != nil {
		return err
	}

	invokeSpec := &peer.ChaincodeInvocationSpec{}
	err = proto.Unmarshal(ccPropPayload.Input, invokeSpec)
	if err != nil {
		return err
	}

	receipt := TxReceipt{
		TransactionHash: string(*param),
		BlockHash:       hex.EncodeToString(blkHeader.Hash()),
		BlockNumber:     strconv.FormatUint(blkHeader.GetNumber(), 10),
	}

	args = invokeSpec.GetChaincodeSpec().GetInput().Args
	// First arg is the callee address. If it is zero address, tx was a contract creation
	callee, err := hex.DecodeString(string(args[0]))
	if err != nil {
		return err
	}

	if bytes.Equal(callee, zeroAddress) {
		receipt.ContractAddress = string(respPayload.GetResponse().GetPayload())
	}
	*reply = receipt

	return nil
}

func Query(chClient apitxn.ChannelClient, chaincodeID string, function string, queryArgs [][]byte) ([]byte, error) {

	return chClient.Query(apitxn.QueryRequest{
		ChaincodeID: chaincodeID,
		Fcn:         function,
		Args:        queryArgs,
	})
}

func Strip0xFromHex(addr string) string {
	stripped := strings.Split(addr, "0x")
	// if len(stripped) != 1 {
	// 	panic("Had more then 1 0x in address")
	// }
	return stripped[len(stripped)-1]
}

func GetPayloads(txActions *peer.TransactionAction) (*peer.ChaincodeProposalPayload, *peer.ChaincodeAction, error) {
	// TODO: pass in the tx type (in what follows we're assuming the type is ENDORSER_TRANSACTION)
	ccPayload := &peer.ChaincodeActionPayload{}
	err := proto.Unmarshal(txActions.Payload, ccPayload)
	if err != nil {
		return nil, nil, err
	}

	if ccPayload.Action == nil || ccPayload.Action.ProposalResponsePayload == nil {
		return nil, nil, fmt.Errorf("no payload in ChaincodeActionPayload")
	}

	ccProposalPayload := &peer.ChaincodeProposalPayload{}
	err = proto.Unmarshal(ccPayload.ChaincodeProposalPayload, ccProposalPayload)
	if err != nil {
		return nil, nil, err
	}

	pRespPayload := &peer.ProposalResponsePayload{}
	err = proto.Unmarshal(ccPayload.Action.ProposalResponsePayload, pRespPayload)
	if err != nil {
		return nil, nil, err
	}

	if pRespPayload.Extension == nil {
		return nil, nil, fmt.Errorf("response payload is missing extension")
	}

	respPayload := &peer.ChaincodeAction{}
	err = proto.Unmarshal(pRespPayload.Extension, respPayload)
	if err != nil {
		return ccProposalPayload, nil, err
	}
	return ccProposalPayload, respPayload, nil
}
