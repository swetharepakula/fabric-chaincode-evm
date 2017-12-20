package ethserver

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type EthRPCService struct {
	EthService
	Web3Service
}
type Web3Service interface {
	ClientVersion(args *EthRPCArgs, reply *string) error
}
type EthService interface {
	GetCode(*getCodeArgs, *string) error
}

type ethRPCService struct{}
type web3Service struct{}

type EthRPCArgs struct{}
type getCodeArgs struct{}

type EthServer struct {
	Server   *rpc.Server
	listener net.Listener
}

func NewEthService() EthService {
	return new(ethRPCService)
}

func NewWeb3Service() Web3Service {
	return new(web3Service)
}

func NewEthServer(eth EthService, web3 Web3Service) *EthServer {
	server := rpc.NewServer()

	ethService := EthRPCService{eth, web3}
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

func (req *web3Service) ClientVersion(args *EthRPCArgs, reply *string) error {
	*reply = "0.0"
	return nil
}
func (req *ethRPCService) GetCode(args *getCodeArgs, reply *string) error {
	return nil
}
