/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabproxy_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/burrow/account"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/execution/evm/events"
	evm_event "github.com/hyperledger/fabric-chaincode-evm/event"
	"github.com/hyperledger/fabric-chaincode-evm/fabproxy"
	"github.com/hyperledger/fabric-chaincode-evm/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var evmcc = "evmcc"
var _ = Describe("Ethservice", func() {
	var (
		ethservice fabproxy.EthService

		mockChClient     *mocks.MockChannelClient
		mockLedgerClient *mocks.MockLedgerClient
		channelID        string
	)

	BeforeEach(func() {
		mockChClient = &mocks.MockChannelClient{}
		mockLedgerClient = &mocks.MockLedgerClient{}
		channelID = "test-channel"

		ethservice = fabproxy.NewEthService(mockChClient, mockLedgerClient, channelID, evmcc)
	})

	Describe("GetCode", func() {
		var (
			sampleCode    []byte
			sampleAddress string
		)

		BeforeEach(func() {
			sampleCode = []byte("sample-code")
			mockChClient.QueryReturns(channel.Response{
				Payload: sampleCode,
			}, nil)

			sampleAddress = "1234567123"
		})

		It("returns the code associated to that address", func() {
			var reply string

			err := ethservice.GetCode(&http.Request{}, &sampleAddress, &reply)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockChClient.QueryCallCount()).To(Equal(1))
			chReq, reqOpts := mockChClient.QueryArgsForCall(0)
			Expect(chReq).To(Equal(channel.Request{
				ChaincodeID: evmcc,
				Fcn:         "getCode",
				Args:        [][]byte{[]byte(sampleAddress)},
			}))

			Expect(reqOpts).To(HaveLen(0))

			Expect(reply).To(Equal(string(sampleCode)))
		})

		Context("when the address has `0x` prefix", func() {
			BeforeEach(func() {
				sampleAddress = "0x123456"
			})
			It("returns the code associated with that address", func() {
				var reply string

				err := ethservice.GetCode(&http.Request{}, &sampleAddress, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockChClient.QueryCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.QueryArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         "getCode",
					Args:        [][]byte{[]byte(sampleAddress[2:])},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal(string(sampleCode)))
			})
		})

		Context("when the ledger errors when processing a query", func() {
			BeforeEach(func() {
				mockChClient.QueryReturns(channel.Response{}, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply string

				err := ethservice.GetCode(&http.Request{}, &sampleAddress, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to query the ledger")))

				Expect(reply).To(BeEmpty())
			})
		})
	})

	Describe("Call", func() {
		var (
			encodedResponse []byte
			sampleArgs      *fabproxy.EthArgs
		)

		BeforeEach(func() {
			sampleResponse := []byte("sample response")
			encodedResponse = make([]byte, hex.EncodedLen(len(sampleResponse)))
			hex.Encode(encodedResponse, sampleResponse)
			mockChClient.QueryReturns(channel.Response{
				Payload: sampleResponse,
			}, nil)

			sampleArgs = &fabproxy.EthArgs{
				To:   "1234567123",
				Data: "sample-data",
			}
		})

		It("returns the value of the simulation of executing a smart contract with a `0x` prefix", func() {

			var reply string

			err := ethservice.Call(&http.Request{}, sampleArgs, &reply)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockChClient.QueryCallCount()).To(Equal(1))
			chReq, reqOpts := mockChClient.QueryArgsForCall(0)
			Expect(chReq).To(Equal(channel.Request{
				ChaincodeID: evmcc,
				Fcn:         sampleArgs.To,
				Args:        [][]byte{[]byte(sampleArgs.Data)},
			}))

			Expect(reqOpts).To(HaveLen(0))

			Expect(reply).To(Equal("0x" + string(encodedResponse)))
		})

		Context("when the ledger errors when processing a query", func() {
			BeforeEach(func() {
				mockChClient.QueryReturns(channel.Response{}, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply string

				err := ethservice.Call(&http.Request{}, &fabproxy.EthArgs{}, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to query the ledger")))
				Expect(reply).To(BeEmpty())
			})
		})

		Context("when the address has a `0x` prefix", func() {
			BeforeEach(func() {
				sampleArgs.To = "0x" + sampleArgs.To
			})
			It("strips the prefix from the query", func() {
				var reply string

				err := ethservice.Call(&http.Request{}, sampleArgs, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockChClient.QueryCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.QueryArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         sampleArgs.To[2:],
					Args:        [][]byte{[]byte(sampleArgs.Data)},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal("0x" + string(encodedResponse)))
			})
		})

		Context("when the data has a `0x` prefix", func() {
			BeforeEach(func() {
				sampleArgs.Data = "0x" + sampleArgs.Data
			})

			It("strips the prefix from the query", func() {
				var reply string

				err := ethservice.Call(&http.Request{}, sampleArgs, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockChClient.QueryCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.QueryArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         sampleArgs.To,
					Args:        [][]byte{[]byte(sampleArgs.Data[2:])},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal("0x" + string(encodedResponse)))
			})
		})
	})

	Describe("SendTransaction", func() {
		var (
			sampleResponse channel.Response
			sampleArgs     *fabproxy.EthArgs
		)

		BeforeEach(func() {
			sampleResponse = channel.Response{
				Payload:       []byte("sample-response"),
				TransactionID: "1",
			}
			mockChClient.ExecuteReturns(sampleResponse, nil)

			sampleArgs = &fabproxy.EthArgs{
				To:   "1234567123",
				Data: "sample-data",
			}
		})

		It("returns the transaction id", func() {
			var reply string
			err := ethservice.SendTransaction(&http.Request{}, sampleArgs, &reply)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockChClient.ExecuteCallCount()).To(Equal(1))
			chReq, reqOpts := mockChClient.ExecuteArgsForCall(0)
			Expect(chReq).To(Equal(channel.Request{
				ChaincodeID: evmcc,
				Fcn:         sampleArgs.To,
				Args:        [][]byte{[]byte(sampleArgs.Data)},
			}))

			Expect(reqOpts).To(HaveLen(0))

			Expect(reply).To(Equal(string(sampleResponse.TransactionID)))
		})

		Context("when the transaction is a contract deployment", func() {
			BeforeEach(func() {
				sampleArgs.To = ""
			})

			It("returns the transaction id", func() {
				var reply string
				err := ethservice.SendTransaction(&http.Request{}, sampleArgs, &reply)
				Expect(err).ToNot(HaveOccurred())

				zeroAddress := hex.EncodeToString(fabproxy.ZeroAddress)
				Expect(mockChClient.ExecuteCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.ExecuteArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         zeroAddress,
					Args:        [][]byte{[]byte(sampleArgs.Data)},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal(string(sampleResponse.TransactionID)))
			})
		})

		Context("when the ledger errors when processing a query", func() {
			BeforeEach(func() {
				mockChClient.ExecuteReturns(channel.Response{}, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply string

				err := ethservice.SendTransaction(&http.Request{}, &fabproxy.EthArgs{}, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to execute transaction")))
				Expect(reply).To(BeEmpty())
			})
		})

		Context("when the address has a `0x` prefix", func() {
			BeforeEach(func() {
				sampleArgs.To = "0x" + sampleArgs.To
			})

			It("strips the prefix before calling the evmscc", func() {
				var reply string
				err := ethservice.SendTransaction(&http.Request{}, sampleArgs, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockChClient.ExecuteCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.ExecuteArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         sampleArgs.To[2:],
					Args:        [][]byte{[]byte(sampleArgs.Data)},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal(string(sampleResponse.TransactionID)))
			})
		})

		Context("when the data has a `0x` prefix", func() {
			BeforeEach(func() {
				sampleArgs.Data = "0x" + sampleArgs.Data
			})

			It("strips the prefix before calling the evmscc", func() {
				var reply string
				err := ethservice.SendTransaction(&http.Request{}, sampleArgs, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockChClient.ExecuteCallCount()).To(Equal(1))
				chReq, reqOpts := mockChClient.ExecuteArgsForCall(0)
				Expect(chReq).To(Equal(channel.Request{
					ChaincodeID: evmcc,
					Fcn:         sampleArgs.To,
					Args:        [][]byte{[]byte(sampleArgs.Data[2:])},
				}))

				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal(string(sampleResponse.TransactionID)))
			})
		})
	})

	FDescribe("GetTransactionReceipt", func() {
		var (
			sampleResponse      channel.Response
			sampleTransaction   *peer.ProcessedTransaction
			otherTransaction    *peer.ProcessedTransaction
			sampleBlock         *common.Block
			sampleTransactionID string
			msg                 events.EventDataLog
			messagePayloads     evm_event.MessagePayloads
			eventPayload        []byte
			eventBytes          []byte
			sampleAddress       string
		)

		BeforeEach(func() {
			sampleResponse = channel.Response{}

			sampleAddress = "82373458164820947891"
			sampleTransactionID = "1234567123"

			var err error
			sampleTransaction, err = GetSampleTransaction([][]byte{[]byte(sampleAddress), []byte("sample arg 2")}, []byte("sample-response"), []byte{}, sampleTransactionID)
			Expect(err).ToNot(HaveOccurred())

			otherTransaction, err = GetSampleTransaction([][]byte{[]byte("1234567"), []byte("sample arg 3")}, []byte("sample-response 2"), []byte{}, "5678")

			sampleBlock = GetSampleBlockWithTransaction(31, []byte("12345abcd"), otherTransaction, sampleTransaction)
			Expect(err).ToNot(HaveOccurred())

			mockLedgerClient.QueryBlockByTxIDReturns(sampleBlock, nil)
			mockLedgerClient.QueryTransactionReturns(sampleTransaction, nil)
		})

		It("returns the transaction receipt associated to that transaction address", func() {
			var reply fabproxy.TxReceipt

			err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockLedgerClient.QueryTransactionCallCount()).To(Equal(1))
			txID, reqOpts := mockLedgerClient.QueryTransactionArgsForCall(0)
			Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
			Expect(reqOpts).To(HaveLen(0))

			Expect(mockLedgerClient.QueryBlockByTxIDCallCount()).To(Equal(1))
			txID, reqOpts = mockLedgerClient.QueryBlockByTxIDArgsForCall(0)
			Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
			Expect(reqOpts).To(HaveLen(0))

			Expect(reply).To(Equal(fabproxy.TxReceipt{
				TransactionHash:   sampleTransactionID,
				TransactionIndex:  "0x1",
				BlockHash:         hex.EncodeToString(sampleBlock.GetHeader().GetDataHash()),
				BlockNumber:       "0x1f",
				GasUsed:           0,
				CumulativeGasUsed: 0,
				To:                "0x" + sampleAddress,
				Status:            string(uint64(1)),
			}))
		})

		Context("when the transaction has associated events", func() {
			BeforeEach(func() {

				var err error
				addr, err := account.AddressFromBytes([]byte(sampleAddress))
				Expect(err).ToNot(HaveOccurred())

				msg = events.EventDataLog{
					Address: addr,
					Topics:  []binary.Word256{[32]byte{0x7, 0x79, 0x9c, 0x56, 0x12, 0x2d, 0x95, 0x24, 0x5a, 0xc7, 0x9c, 0xa1, 0x71, 0xa8, 0xd0, 0x25, 0xdc, 0x20, 0x33, 0x2c, 0xcf, 0xf9, 0x54, 0x8, 0xde, 0x17, 0xbc, 0xaa, 0x73, 0xc8, 0xca, 0x1c}, [32]byte{0xec, 0xa6, 0x62, 0xca, 0xe7, 0x47, 0xb4, 0x67, 0x82, 0x2a, 0x1d, 0x79, 0xb1, 0xeb, 0x1a, 0xee, 0xf1, 0x3b, 0xff, 0x8c, 0x77, 0x39, 0x44, 0x34, 0x46, 0xd4, 0xfd, 0x74, 0xfb, 0x15, 0x12, 0x5f}},
					Data:    []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10},
					Height:  0,
				}
				messagePayloads.Payloads = make([]evm_event.MessagePayload, 0)
				messagePayloads.Payloads = append(messagePayloads.Payloads, evm_event.MessagePayload{Message: msg})
				eventPayload, err = json.Marshal(messagePayloads)
				Expect(err).ToNot(HaveOccurred())

				chaincodeEvent := peer.ChaincodeEvent{
					ChaincodeId: "qscc",
					TxId:        sampleTransactionID,
					EventName:   "Chaincode event",
					Payload:     eventPayload,
				}

				eventBytes, err = proto.Marshal(&chaincodeEvent)
				Expect(err).ToNot(HaveOccurred())

				tx, err := GetSampleTransaction([][]byte{[]byte(sampleAddress), []byte("sample arg 2")}, []byte("sample-response"), []byte{}, sampleTransactionID)
				*sampleTransaction = *tx
				Expect(err).ToNot(HaveOccurred())

				*sampleBlock = *GetSampleBlockWithTransaction(31, []byte("12345abcd"), sampleTransaction, otherTransaction)
			})

			It("returns the transaction receipt associated to that transaction address", func() {
				var reply fabproxy.TxReceipt

				err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockLedgerClient.QueryTransactionCallCount()).To(Equal(1))
				txID, reqOpts := mockLedgerClient.QueryTransactionArgsForCall(0)
				Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
				Expect(reqOpts).To(HaveLen(0))

				Expect(mockLedgerClient.QueryBlockByTxIDCallCount()).To(Equal(1))
				txID, reqOpts = mockLedgerClient.QueryBlockByTxIDArgsForCall(0)
				Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
				Expect(reqOpts).To(HaveLen(0))

				var topics []string
				topics = make([]string, 0)
				for _, topic := range msg.Topics {
					topics = append(topics, topic.String())
				}

				expectedLog := fabproxy.Log{
					Address:     hex.EncodeToString([]byte(sampleAddress)),
					Topics:      topics,
					Data:        string(msg.Data),
					BlockNumber: "0x1f",
					TxHash:      sampleTransactionID,
					//TxIndex: ,
					BlockHash: hex.EncodeToString(sampleBlock.GetHeader().GetDataHash()),
					Index:     string(0),
					Type:      "mined",
				}

				var expectedLogs []fabproxy.Log
				expectedLogs = make([]fabproxy.Log, 0)
				expectedLogs = append(expectedLogs, expectedLog)

				var expectedBloom fabproxy.Bloom
				expectedBloom = fabproxy.CreateBloom(expectedLogs)

				Expect(reply).To(Equal(fabproxy.TxReceipt{
					TransactionHash:   sampleTransactionID,
					TransactionIndex:  "0x0",
					BlockHash:         hex.EncodeToString(sampleBlock.GetHeader().GetDataHash()),
					BlockNumber:       "0x1f",
					GasUsed:           0,
					CumulativeGasUsed: 0,
					To:                "0x82373458",
					Logs:              expectedLogs,
					LogsBloom:         expectedBloom,
					Status:            string(uint64(1)),
				}))
			})

		})

		Context("when the transaction is creation of a smart contract", func() {
			var contractAddress []byte
			BeforeEach(func() {
				contractAddress = []byte("0x123456789abcdef1234")
				zeroAddress := make([]byte, hex.EncodedLen(len(fabproxy.ZeroAddress)))
				hex.Encode(zeroAddress, fabproxy.ZeroAddress)

				tx, err := GetSampleTransaction([][]byte{zeroAddress, []byte("sample arg 2")}, contractAddress, []byte{}, sampleTransactionID)
				*sampleTransaction = *tx
				Expect(err).ToNot(HaveOccurred())

				*sampleBlock = *GetSampleBlockWithTransaction(31, []byte("12345abcd"), sampleTransaction, otherTransaction)
			})

			It("returns the contract address in the transaction receipt", func() {
				var reply fabproxy.TxReceipt

				err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockLedgerClient.QueryTransactionCallCount()).To(Equal(1))
				txID, reqOpts := mockLedgerClient.QueryTransactionArgsForCall(0)
				Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
				Expect(reqOpts).To(HaveLen(0))

				Expect(mockLedgerClient.QueryBlockByTxIDCallCount()).To(Equal(1))
				txID, reqOpts = mockLedgerClient.QueryBlockByTxIDArgsForCall(0)
				Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID)))
				Expect(reqOpts).To(HaveLen(0))

				Expect(reply).To(Equal(fabproxy.TxReceipt{
					TransactionHash:   sampleTransactionID,
					TransactionIndex:  "0x0",
					BlockHash:         hex.EncodeToString(sampleBlock.GetHeader().GetDataHash()),
					BlockNumber:       "0x1f",
					ContractAddress:   string(contractAddress),
					GasUsed:           0,
					CumulativeGasUsed: 0,
					Logs:              nil,
					LogsBloom:         fabproxy.CreateBloom(nil),
					Status:            string(uint64(1)),
				}))
			})

			Context("when transaction ID has `0x` prefix", func() {
				BeforeEach(func() {
					sampleTransactionID = "0x" + sampleTransactionID
				})
				It("strips the prefix before querying the ledger", func() {
					var reply fabproxy.TxReceipt

					err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
					Expect(err).ToNot(HaveOccurred())

					Expect(mockLedgerClient.QueryTransactionCallCount()).To(Equal(1))
					txID, reqOpts := mockLedgerClient.QueryTransactionArgsForCall(0)
					Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID[2:])))
					Expect(reqOpts).To(HaveLen(0))

					Expect(mockLedgerClient.QueryBlockByTxIDCallCount()).To(Equal(1))
					txID, reqOpts = mockLedgerClient.QueryBlockByTxIDArgsForCall(0)
					Expect(txID).To(Equal(fab.TransactionID(sampleTransactionID[2:])))
					Expect(reqOpts).To(HaveLen(0))

					Expect(reply).To(Equal(fabproxy.TxReceipt{
						TransactionHash:   sampleTransactionID,
						TransactionIndex:  "0x0",
						BlockHash:         hex.EncodeToString(sampleBlock.GetHeader().GetDataHash()),
						BlockNumber:       "0x1f",
						ContractAddress:   string(contractAddress),
						GasUsed:           0,
						CumulativeGasUsed: 0,
						Logs:              nil,
						LogsBloom:         fabproxy.CreateBloom(nil),
						Status:            string(uint64(1)),
					}))
				})
			})
		})

		Context("when the ledger errors when processing a transaction query for the transaction", func() {
			BeforeEach(func() {
				mockLedgerClient.QueryTransactionReturns(nil, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply fabproxy.TxReceipt

				err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to query the ledger")))
				Expect(reply).To(BeZero())
			})
		})

		Context("when the ledger errors when processing a query for the block", func() {
			BeforeEach(func() {
				mockLedgerClient.QueryBlockByTxIDReturns(nil, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply fabproxy.TxReceipt

				err := ethservice.GetTransactionReceipt(&http.Request{}, &sampleTransactionID, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to query the ledger")))
				Expect(reply).To(BeZero())
			})
		})
	})

	Describe("Accounts", func() {
		var (
			sampleAccount string
			arg           string
		)

		BeforeEach(func() {
			sampleAccount = "123456ABCD"
			mockChClient.QueryReturns(channel.Response{
				Payload: []byte(sampleAccount),
			}, nil)

		})

		It("requests the user address from the evmscc based on the user cert", func() {
			var reply []string

			err := ethservice.Accounts(&http.Request{}, &arg, &reply)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockChClient.QueryCallCount()).To(Equal(1))
			chReq, reqOpts := mockChClient.QueryArgsForCall(0)
			Expect(chReq).To(Equal(channel.Request{
				ChaincodeID: evmcc,
				Fcn:         "account",
				Args:        [][]byte{},
			}))

			Expect(reqOpts).To(HaveLen(0))
			expectedResponse := []string{"0x" + strings.ToLower(sampleAccount)}
			Expect(reply).To(Equal(expectedResponse))
		})

		Context("when the ledger errors when processing a query", func() {
			BeforeEach(func() {
				mockChClient.QueryReturns(channel.Response{}, errors.New("boom!"))
			})

			It("returns a corresponding error", func() {
				var reply []string
				err := ethservice.Accounts(&http.Request{}, &arg, &reply)
				Expect(err).To(MatchError(ContainSubstring("Failed to query the ledger")))
				Expect(reply).To(BeEmpty())
			})
		})
	})

	Describe("EstimateGas", func() {
		It("always returns zero", func() {
			var reply string
			err := ethservice.EstimateGas(&http.Request{}, &fabproxy.EthArgs{}, &reply)
			Expect(err).ToNot(HaveOccurred())
			Expect(reply).To(Equal("0x0"))
		})
	})

	Describe("GetBalance", func() {
		It("always returns zero", func() {
			arg := make([]string, 2)
			var reply string
			err := ethservice.GetBalance(&http.Request{}, &arg, &reply)
			Expect(err).ToNot(HaveOccurred())
			Expect(reply).To(Equal("0x0"))
		})
	})

	Describe("GetBlockByNumber", func() {
		Context("when provided with bad parameters", func() {
			var reply fabproxy.Block

			It("returns an error when arg length is not 2", func() {
				var arg []interface{}
				err := ethservice.GetBlockByNumber(&http.Request{}, &arg, &reply)
				Expect(err).To(HaveOccurred())
			})

			It("returns an error when the first arg is not a string", func() {
				arg := make([]interface{}, 2)
				arg[0] = false
				err := ethservice.GetBlockByNumber(&http.Request{}, &arg, &reply)
				Expect(err).To(HaveOccurred())
			})
			It("returns an error when first arg is not a named block or numbered block", func() {
				arg := make([]interface{}, 2)
				arg[0] = "hurf%&"
				err := ethservice.GetBlockByNumber(&http.Request{}, &arg, &reply)
				Expect(err).To(HaveOccurred())
			})

			It("returns an error, when the second arg is not a booleand", func() {
				arg := make([]interface{}, 2)
				arg[0] = "latest"
				arg[1] = "durf"
				err := ethservice.GetBlockByNumber(&http.Request{}, &arg, &reply)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when there are good parameters", func() {
			var (
				reply                fabproxy.Block
				args                 []interface{}
				fullTransactions     bool
				requestedBlockNumber string
			)

			JustBeforeEach(func() {
				args = make([]interface{}, 2)
				args[0] = requestedBlockNumber
				args[1] = fullTransactions
			})

			Context("when asking for partial transactions", func() {
				BeforeEach(func() {
					fullTransactions = false
				})

				Context("returns an error when querying the ledger info results in an error", func() {
					BeforeEach(func() {
						requestedBlockNumber = "latest"
						mockLedgerClient.QueryInfoReturns(nil, fmt.Errorf("no block info"))
					})

					It("returns a no blockchain info error when requesting a named block", func() {
						err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
						Expect(err).To(MatchError(ContainSubstring("no block info")))
					})
				})

				Context("when querying the ledger for a block results in an error", func() {
					BeforeEach(func() {
						requestedBlockNumber = "0xa"
						mockLedgerClient.QueryBlockReturns(nil, fmt.Errorf("no block"))
					})

					It("returns the error", func() {
						err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
						Expect(err).To(HaveOccurred())
					})

					Context("when querying for a named block", func() {
						BeforeEach(func() {
							requestedBlockNumber = "latest"
							mockLedgerClient.QueryInfoReturns(&fab.BlockchainInfoResponse{BCI: &common.BlockchainInfo{Height: 1}}, nil)
						})

						It("returns an error", func() {
							err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
							Expect(err).To(HaveOccurred())
						})
					})
				})

				It("returns an error when asked for pending blocks", func() {
					args[0] = "pending"
					err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
					Expect(err).To(HaveOccurred())
				})

				Context("when a block is requested by number", func() {
					var uintBlockNumber uint64

					BeforeEach(func() {
						requestedBlockNumber = "abc0"

						var err error
						uintBlockNumber, err = strconv.ParseUint(requestedBlockNumber, 16, 64)
						Expect(err).ToNot(HaveOccurred())
					})

					It("requests a block by number", func() {
						sampleBlock := GetSampleBlock(uintBlockNumber, []byte("def\xFF"))
						mockLedgerClient.QueryBlockReturns(sampleBlock, nil)

						err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
						Expect(err).ToNot(HaveOccurred())

						Expect(reply.Number).To(Equal("0x"+requestedBlockNumber), "block number")
						Expect(reply.Hash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.DataHash)), "block data hash")
						Expect(reply.ParentHash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.PreviousHash)), "block parent hash")
						txns := reply.Transactions
						Expect(txns).To(HaveLen(2))
						Expect(txns[0]).To(BeEquivalentTo("0x5678"))
						Expect(txns[1]).To(BeEquivalentTo("0x1234"))
					})
				})

				Context("when the block is requested by name", func() {
					var uintBlockNumber uint64

					BeforeEach(func() {
						requestedBlockNumber = "latest"

						var err error
						uintBlockNumber, err = strconv.ParseUint("abc0", 16, 64)
						Expect(err).ToNot(HaveOccurred())

						mockLedgerClient.QueryInfoReturns(&fab.BlockchainInfoResponse{BCI: &common.BlockchainInfo{Height: uintBlockNumber + 1}}, nil)
					})

					It("returns the block", func() {
						sampleBlock := GetSampleBlock(uintBlockNumber, []byte("def\xFF"))
						mockLedgerClient.QueryBlockReturns(sampleBlock, nil)

						err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
						Expect(err).ToNot(HaveOccurred())
						Expect(reply.Number).To(Equal("0xabc0"), "block number")
						Expect(reply.Hash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.DataHash)), "block data hash")
						Expect(reply.ParentHash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.PreviousHash)), "block parent hash")
						txns := reply.Transactions
						Expect(txns).To(HaveLen(2))
						Expect(txns[0]).To(BeEquivalentTo("0x5678"))
						Expect(txns[1]).To(BeEquivalentTo("0x1234"))
					})
				})
			})

			Context("when asking for full transactions", func() {
				var uintBlockNumber uint64
				BeforeEach(func() {
					requestedBlockNumber = "abc0"
					fullTransactions = true

					var err error
					uintBlockNumber, err = strconv.ParseUint("abc0", 16, 64)
					Expect(err).ToNot(HaveOccurred())

					mockLedgerClient.QueryInfoReturns(&fab.BlockchainInfoResponse{BCI: &common.BlockchainInfo{Height: uintBlockNumber + 1}}, nil)
				})

				It("returns a block with transactions with detail", func() {
					sampleBlock := GetSampleBlock(uintBlockNumber, []byte("def\xFF"))
					mockLedgerClient.QueryBlockReturns(sampleBlock, nil)

					err := ethservice.GetBlockByNumber(&http.Request{}, &args, &reply)
					Expect(err).ToNot(HaveOccurred())

					blockNumber := "0x" + requestedBlockNumber
					Expect(reply.Number).To(Equal(blockNumber), "block number")

					blockHash := "0x" + hex.EncodeToString(sampleBlock.Header.DataHash)
					Expect(reply.Hash).To(Equal(blockHash), "block data hash")
					Expect(reply.ParentHash).To(Equal("0x"+hex.EncodeToString(sampleBlock.Header.PreviousHash)), "block parent hash")

					txns := reply.Transactions
					Expect(txns).To(HaveLen(2))

					t0, ok := txns[0].(fabproxy.Transaction)
					Expect(ok).To(BeTrue())
					Expect(t0.BlockHash).To(Equal(blockHash))
					Expect(t0.BlockNumber).To(Equal(blockNumber))
					Expect(t0.To).To(Equal("0x12345678"))
					Expect(t0.Input).To(Equal("0xsample arg 1"))
					Expect(t0.TransactionIndex).To(Equal("0x0"))
					Expect(t0.Hash).To(Equal("0x5678"))

					t1, ok := txns[1].(fabproxy.Transaction)
					Expect(ok).To(BeTrue())
					Expect(t1.BlockHash).To(Equal(blockHash))
					Expect(t1.BlockNumber).To(Equal(blockNumber))
					Expect(t1.To).To(Equal("0x98765432"))
					Expect(t1.Input).To(Equal("0xsample arg 2"))
					Expect(t1.TransactionIndex).To(Equal("0x1"))
					Expect(t1.Hash).To(Equal("0x1234"))
				})
			})
		})
	})

	Describe("GetTransactionByHash", func() {
		var reply fabproxy.Transaction

		It("returns an error when given an empty string for transaction hash", func() {
			txID := ""
			err := ethservice.GetTransactionByHash(&http.Request{}, &txID, &reply)
			Expect(err).To(HaveOccurred())
		})

		Context("if the ledger returns an error", func() {
			BeforeEach(func() {
				mockLedgerClient.QueryBlockByTxIDReturns(nil, fmt.Errorf("bad ledger lookup"))
			})
			It("returns an error ", func() {
				txID := "0x1234"
				err := ethservice.GetTransactionByHash(&http.Request{}, &txID, &reply)
				Expect(err).To(HaveOccurred())
			})
		})

		It("gets a transaction", func() {
			txID := "0x1234"
			block := GetSampleBlock(1, []byte("def\xFF"))
			mockLedgerClient.QueryBlockByTxIDReturns(block, nil)
			err := ethservice.GetTransactionByHash(&http.Request{}, &txID, &reply)
			Expect(err).ToNot(HaveOccurred())
			Expect(reply.Hash).To(Equal(txID), "txn id hash that was passed in")
			Expect(reply.BlockHash).To(Equal("0x"+hex.EncodeToString(block.Header.DataHash)), "block data hash")
			Expect(reply.BlockNumber).To(Equal("0x1"), "blocknumber")
			Expect(reply.TransactionIndex).To(Equal("0x1"), "txn Index")
			Expect(reply.To).To(Equal("0x98765432"))
			Expect(reply.Input).To(Equal("0xsample arg 2"))
		})
	})
})

func GetSampleBlock(blockNumber uint64, blkHash []byte) *common.Block {
	tx, err := GetSampleTransaction([][]byte{[]byte("12345678"), []byte("sample arg 1")}, []byte("sample-response1"), []byte{}, "5678")
	Expect(err).ToNot(HaveOccurred())
	txn1, err := proto.Marshal(tx.TransactionEnvelope)
	Expect(err).ToNot(HaveOccurred())

	tx, err = GetSampleTransaction([][]byte{[]byte("98765432"), []byte("sample arg 2")}, []byte("sample-response2"), []byte{}, "1234")
	txn2, err := proto.Marshal(tx.TransactionEnvelope)
	Expect(err).ToNot(HaveOccurred())

	phash := []byte("abc\x00")
	return &common.Block{
		Header: &common.BlockHeader{Number: blockNumber,
			PreviousHash: phash,
			DataHash:     blkHash},
		Data: &common.BlockData{Data: [][]byte{txn1, txn2}},
	}
}

func GetSampleBlockWithTransaction(blockNumber uint64, blkHash []byte, txns ...*peer.ProcessedTransaction) *common.Block {

	blockData := [][]byte{}

	for _, tx := range txns {
		txn, err := proto.Marshal(tx.TransactionEnvelope)
		Expect(err).ToNot(HaveOccurred())

		blockData = append(blockData, txn)
	}

	phash := []byte("abc\x00")
	return &common.Block{
		Header: &common.BlockHeader{Number: blockNumber,
			PreviousHash: phash,
			DataHash:     blkHash},
		Data: &common.BlockData{Data: blockData},
	}
}

func GetSampleTransaction(inputArgs [][]byte, txResponse, eventBytes []byte, txId string) (*peer.ProcessedTransaction, error) {

	respPayload := &peer.ChaincodeAction{
		Events: eventBytes,
		Response: &peer.Response{
			Payload: txResponse,
		},
	}

	ext, err := proto.Marshal(respPayload)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	pRespPayload := &peer.ProposalResponsePayload{
		Extension: ext,
	}

	ccProposalPayload, err := proto.Marshal(pRespPayload)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	invokeSpec := &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{
				Name: evmcc,
			},
			Input: &peer.ChaincodeInput{
				Args: inputArgs,
			},
		},
	}

	invokeSpecBytes, err := proto.Marshal(invokeSpec)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	ccPropPayload, err := proto.Marshal(&peer.ChaincodeProposalPayload{
		Input: invokeSpecBytes,
	})
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	ccPayload := &peer.ChaincodeActionPayload{
		Action: &peer.ChaincodeEndorsedAction{
			ProposalResponsePayload: ccProposalPayload,
		},
		ChaincodeProposalPayload: ccPropPayload,
	}

	actionPayload, err := proto.Marshal(ccPayload)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	txAction := &peer.TransactionAction{
		Payload: actionPayload,
	}

	txActions := &peer.Transaction{
		Actions: []*peer.TransactionAction{txAction},
	}

	actionsPayload, err := proto.Marshal(txActions)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	chdr := &common.ChannelHeader{TxId: txId}
	chdrBytes, err := proto.Marshal(chdr)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	payload := &common.Payload{
		Header: &common.Header{
			ChannelHeader: chdrBytes,
		},
		Data: actionsPayload,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return &peer.ProcessedTransaction{}, err
	}

	tx := &peer.ProcessedTransaction{
		TransactionEnvelope: &common.Envelope{
			Payload: payloadBytes,
		},
	}

	return tx, nil
}
