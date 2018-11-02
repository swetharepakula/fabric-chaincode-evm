/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package event_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hyperledger/burrow/account"
	"github.com/hyperledger/burrow/event"
	"github.com/hyperledger/burrow/execution/evm/events"
	evm_event "github.com/hyperledger/fabric-chaincode-evm/event"
	"github.com/hyperledger/fabric-chaincode-evm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event", func() {

	var (
		eventManager evm_event.EventManager
		mockStub     *mocks.MockStub
		addr         account.Address
		publisher    event.Publisher
	)

	BeforeEach(func() {
		mockStub = &mocks.MockStub{}
		eventManager = *evm_event.NewEventManager(mockStub, publisher)

		var err error
		addr, err = account.AddressFromBytes([]byte("0000000000000address"))
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Publish", func() {
		var (
			ctx     context.Context
			message events.EventDataLog
			tags    map[string]interface{}
		)

		BeforeEach(func() {
			ctx = context.Background()
			message = events.EventDataLog{
				Address: addr,
				Height:  0,
			}
			tags = map[string]interface{}{"EventID": fmt.Sprintf("Log/%s", addr)}
		})

		Context("when an event is emitted by calling the publish function", func() {
			Context("if it is a log event", func() {
				It("appends the new message info into the eventCache", func() {
					originalLength := len(eventManager.EventCache)
					err := eventManager.Publish(ctx, &message, tags)
					Expect(err).ToNot(HaveOccurred())
					newLength := len(eventManager.EventCache)
					Expect(newLength).To(Equal(originalLength + 1))
					messageInfo := evm_event.MessageInfo{
						Ctx:     ctx,
						Message: message,
						Tags:    tags,
					}
					Expect(eventManager.EventCache[newLength-1]).To(Equal(messageInfo))
				})
			})

			Context("if it is not a log event", func() {
				It("does nothing", func() {
					originalLength := len(eventManager.EventCache)
					originalEventCache := eventManager.EventCache
					var alt_tags map[string]interface{}
					alt_tags = map[string]interface{}{"EventID": fmt.Sprintf("Acc/%s/Call", addr)}
					err := eventManager.Publish(ctx, &message, alt_tags)
					Expect(err).ToNot(HaveOccurred())
					newLength := len(eventManager.EventCache)
					newEventCache := eventManager.EventCache
					Expect(newLength).To(Equal(originalLength))
					Expect(newEventCache).To(Equal(originalEventCache))
				})
			})
		})

		Context("when there is a type mismatch in the event ID tag", func() {
			It("an error occurs", func() {
				var err_tags map[string]interface{}
				err_tags = map[string]interface{}{"EventID": []byte(fmt.Sprintf("Log/%s", addr))}
				err := eventManager.Publish(ctx, &message, err_tags)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when there is a type mismatch in the message type", func() {
			It("an error occurs", func() {
				err := eventManager.Publish(ctx, message, tags) //passing events.EventDataLog instead of *events.EventDataLog
				Expect(err).To(HaveOccurred())

				msg1 := make(chan events.EventDataLog) //passing chan events.EventDataLog instead of *events.EventDataLog
				err = eventManager.Publish(ctx, msg1, tags)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Flush", func() {
		var (
			ctx      context.Context
			message1 events.EventDataLog
			message2 events.EventDataLog
			tags     map[string]interface{}
		)

		BeforeEach(func() {
			ctx = context.Background()
			message1 = events.EventDataLog{
				Address: addr,
				Height:  0,
			}
			message2 = events.EventDataLog{
				Address: addr,
				Height:  1,
			}
			tags = map[string]interface{}{"EventID": fmt.Sprintf("Log/%s", addr)}
		})

		Context("when a single event is emitted", func() {
			It("sets a new event with a single messageInfo object payload", func() {
				err := eventManager.Publish(ctx, &message1, tags)
				Expect(err).ToNot(HaveOccurred())
				er := eventManager.Flush("Chaincode event")
				Expect(er).ToNot(HaveOccurred())

				var messagePayloads1 evm_event.MessagePayloads
				messagePayloads1.Payloads = make([]evm_event.MessagePayload, 0)
				messagePayloads1.Payloads = append(messagePayloads1.Payloads, evm_event.MessagePayload{Message: message1})
				expectedPayload1, err1 := json.Marshal(messagePayloads1)
				Expect(err1).ToNot(HaveOccurred())

				Expect(mockStub.SetEventCallCount()).To(Equal(1))
				setEventName, setEventPayload := mockStub.SetEventArgsForCall(0)
				Expect(setEventName).To(Equal("Chaincode event"))
				Expect(setEventPayload).To(Equal(expectedPayload1))

				var unmarshaledPayloads evm_event.MessagePayloads
				e := json.Unmarshal(setEventPayload, &unmarshaledPayloads)
				Expect(e).ToNot(HaveOccurred())
				Expect(unmarshaledPayloads.Payloads[0].Message).To(Equal(message1))
				Expect(unmarshaledPayloads).To(Equal(messagePayloads1))
			})
		})

		Context("when multiple events are emitted", func() {
			It("sets a new event with a payload consisting of messageInfo objects marshaled together", func() {
				err := eventManager.Publish(ctx, &message1, tags)
				Expect(err).ToNot(HaveOccurred())
				err1 := eventManager.Publish(ctx, &message2, tags)
				Expect(err1).ToNot(HaveOccurred())
				er := eventManager.Flush("Chaincode event")
				Expect(er).ToNot(HaveOccurred())

				var messagePayloads2 evm_event.MessagePayloads
				messagePayloads2.Payloads = make([]evm_event.MessagePayload, 0)
				messagePayloads2.Payloads = append(messagePayloads2.Payloads, evm_event.MessagePayload{Message: message1})
				messagePayloads2.Payloads = append(messagePayloads2.Payloads, evm_event.MessagePayload{Message: message2})
				expectedPayload2, err2 := json.Marshal(messagePayloads2)
				Expect(err2).ToNot(HaveOccurred())

				Expect(mockStub.SetEventCallCount()).To(Equal(1))
				setEventName, setEventPayload := mockStub.SetEventArgsForCall(0)
				Expect(setEventName).To(Equal("Chaincode event"))
				Expect(setEventPayload).To(Equal(expectedPayload2))

				var unmarshaledPayloads evm_event.MessagePayloads
				e := json.Unmarshal(setEventPayload, &unmarshaledPayloads)
				Expect(e).ToNot(HaveOccurred())
				Expect(unmarshaledPayloads.Payloads[0].Message).To(Equal(message1))
				Expect(unmarshaledPayloads.Payloads[1].Message).To(Equal(message2))
				Expect(unmarshaledPayloads).To(Equal(messagePayloads2))
			})
		})

		Context("when the event name is invalid (nil string)", func() {
			BeforeEach(func() {
				mockStub.SetEventReturns(errors.New("error: nil event name"))
			})

			It("returns an error", func() {
				err := eventManager.Publish(ctx, &message1, tags)
				Expect(err).ToNot(HaveOccurred())
				err1 := eventManager.Publish(ctx, &message2, tags)
				Expect(err1).ToNot(HaveOccurred())
				er := eventManager.Flush("")
				Expect(er).To(HaveOccurred())
			})
		})
	})
})
