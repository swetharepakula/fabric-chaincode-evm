/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/burrow/event"
	"github.com/hyperledger/burrow/execution/evm/events"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type MessageInfo struct {
	Ctx     context.Context        `json:"ctx"`
	Message events.EventDataLog    `json:"message"`
	Tags    map[string]interface{} `json:"tags"`
}

type MessagePayload struct {
	Message events.EventDataLog
}

type MessagePayloads struct {
	Payloads []MessagePayload `json:"payloads"`
}

type EventManager struct {
	stub       shim.ChaincodeStubInterface
	EventCache []MessageInfo
	publisher  event.Publisher
}

func NewEventManager(stub shim.ChaincodeStubInterface, publisher event.Publisher) *EventManager {
	return &EventManager{
		stub:       stub,
		EventCache: make([]MessageInfo, 0),
		publisher:  publisher,
	}
}

func (evmgr *EventManager) Flush(eventName string) error {
	var err error
	var eventMsgs MessagePayloads
	eventMsgs.Payloads = make([]MessagePayload, 0)

	if len(evmgr.EventCache) > 0 {
		for i := 0; i < len(evmgr.EventCache); i++ {
			eventDataLog := evmgr.EventCache[i].Message
			msg := MessagePayload{Message: eventDataLog}
			eventMsgs.Payloads = append(eventMsgs.Payloads, msg)
		}

		payload, er := json.Marshal(eventMsgs)
		//I am not sure whether this will ever give an error...
		if er != nil {
			return fmt.Errorf("Failed to marshal event messages: %s", er.Error())
		}
		err = evmgr.stub.SetEvent(eventName, payload)
		return err
	}

	return nil
}

func (evmgr *EventManager) Publish(ctx context.Context, message interface{}, tags map[string]interface{}) error {
	evID, ok := tags["EventID"].(string)
	if !ok {
		return fmt.Errorf("type mismatch: expected string but received %T", tags["EventID"])
	}

	msg, ok1 := message.(*events.EventDataLog)
	if !ok1 {
		return fmt.Errorf("type mismatch: expected *events.EventDataLog but received %T", message)
	}

	//Burrow EVM emits other events related to state (such as account call) as well, but we are only interested in log events
	if evID[0:3] == "Log" {
		evmgr.EventCache = append(evmgr.EventCache, MessageInfo{
			Ctx:     ctx,
			Message: *msg,
			Tags:    tags,
		})
	}
	return nil
}
