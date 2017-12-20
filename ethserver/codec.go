package ethserver

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strings"
)

type rpcCodec struct {
	codec rpc.ServerCodec
}

type httpConn struct {
	in  io.Reader
	out io.Writer
}

func (c *httpConn) Read(data []byte) (n int, err error)  { return c.in.Read(data) }
func (c *httpConn) Write(data []byte) (n int, err error) { return c.out.Write(data) }
func (c *httpConn) Close() error                         { return nil }

func NewRPCCodec(r *http.Request, w http.ResponseWriter) rpc.ServerCodec {
	return &rpcCodec{
		codec: jsonrpc.NewServerCodec(&httpConn{in: r.Body, out: w}),
	}
}

func (c *rpcCodec) ReadRequestHeader(req *rpc.Request) error {
	err := c.codec.ReadRequestHeader(req)
	if err != nil {
		return err
	}
	serviceMethod := strings.Split(req.ServiceMethod, "_")
	service := "EthRPCService"
	var method string

	switch serviceMethod[0] {
	case "web3":
		method = strings.Title(serviceMethod[len(serviceMethod)-1])
	case "eth":
		method = strings.Title(serviceMethod[len(serviceMethod)-1])
	default:
		return errors.New("Service not found")
	}
	req.ServiceMethod = fmt.Sprintf("%s.%s", service, method)

	return nil
}

func (c *rpcCodec) ReadRequestBody(body interface{}) error {
	return c.codec.ReadRequestBody(body)
}

func (c *rpcCodec) WriteResponse(res *rpc.Response, body interface{}) error {
	return c.codec.WriteResponse(res, body)
}

func (c *rpcCodec) Close() error {
	return c.codec.Close()
}
