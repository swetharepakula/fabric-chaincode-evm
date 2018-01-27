package ethserver

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric/protos/peer"
)

type EthRPCService struct {
	EthService
}

type EthService interface {
	GetCode(*GetCodeArgs, *string) error
}

type ethRPCService struct {
	sdk *fabsdk.FabricSDK
}

type EthRPCArgs struct{}
type GetCodeArgs string

type EthServer struct {
	Server   *rpc.Server
	listener net.Listener
}

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
	server.Register(ethService)

	return &EthServer{
		Server: server,
	}
}

func (s *EthServer) Start(port int) {
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	s.listener = l

	http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rpcCodec := NewRPCCodec(r, w)
		w.Header().Set("Content-type", "application/json")
		err := s.Server.ServeRequest(rpcCodec)
		if err != nil {
			errMsg := fmt.Sprintf("Error while serving JSON request, %s", err.Error())
			http.Error(w, errMsg, 500)
		}
		w.WriteHeader(200)
	}))
}

func (s *EthServer) Stop() {
	s.listener.Close()
}

func (req *ethRPCService) GetCode(args *GetCodeArgs, reply *string) error {

	defaultUser := "User1"
	channelID := "mychannel"

	chClient, err := req.sdk.NewChannelClient(channelID, defaultUser)
	if err != nil {
		log.Panic("error creating client", err)
	}

	defer chClient.Close()

	queryArgs := [][]byte{[]byte(channelID), []byte(*args)}

	value, err := chClient.Query(apitxn.QueryRequest{
		ChaincodeID: "lscc",
		Fcn:         "getdepspec",
		Args:        queryArgs,
	})
	if err != nil {
		fmt.Printf("Failed to query: %s\n", err)
	}

	cds := &peer.ChaincodeDeploymentSpec{}
	err = proto.Unmarshal(value, cds)
	if err != nil {
		log.Fatalf("Failed to unmarshal code: %s", err)
	}

	*reply = string(cds.CodePackage)

	return nil
}
