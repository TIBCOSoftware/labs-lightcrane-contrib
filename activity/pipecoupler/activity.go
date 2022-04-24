/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package pipecoupler

import (
	ctx "context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

var log = logger.GetLogger("tibco-pipecoupler")

var initialized bool = false

const (
	DownstreamHost = "DownstreamHost"
	Port           = "Port"
	iTimeout       = "Timeout"
	iData          = "Data"
	oReply         = "Reply"
)

type PipecouplerActivity struct {
	metadata *activity.Metadata
	//context         ctx.Context
	activityToModel map[string]string
	clients         map[string]PipeCouplerClient
	mux             sync.Mutex
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aCMLPipelineActivity := &PipecouplerActivity{
		metadata:        metadata,
		activityToModel: make(map[string]string),
		clients:         make(map[string]PipeCouplerClient),
	}

	return aCMLPipelineActivity
}

func (a *PipecouplerActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *PipecouplerActivity) Eval(context activity.Context) (done bool, err error) {

	log.Debug("[PipecouplerActivity:Eval] entering ........ ")

	client, err := a.getPipeline(context)

	if nil != err {
		return false, err
	}

	timeout, ok := context.GetInput(iTimeout).(int)
	if !ok {
		timeout = 30
	}

	dataMap, ok := context.GetInput(iData).(*data.ComplexObject).Value.(map[string]interface{})
	if !ok {
		log.Warn("No data comes in ... ")
	}
	log.Debug("[PipecouplerActivity:Eval] Input data: ", dataMap)

	var sender string
	if nil != dataMap["Sender"] {
		sender = dataMap["Sender"].(string)
	}
	var id string
	if nil != dataMap["ID"] {
		id = dataMap["ID"].(string)
	}
	var content string
	if nil != dataMap["Content"] {
		content = dataMap["Content"].(string)
	}

	//reqContext, cancel := ctx.WithDeadline(context.Background(), time.Now().Add(time.Duration(timeout)*time.Second))
	reqContext, cancel := ctx.WithTimeout(ctx.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	replyObj, err := client.HandleData(
		reqContext,
		&Data{
			Sender:  sender,
			ID:      id,
			Content: content,
		},
	)
	if err != nil {
		log.Errorf("Error from down stream : %v", err)
	}

	reply := map[string]interface{}{
		"Sender":  replyObj.GetSender(),
		"ID":      replyObj.GetID(),
		"Content": replyObj.GetContent(),
		"Status":  replyObj.GetStatus(),
	}

	context.SetOutput(oReply, &data.ComplexObject{Metadata: "Reply", Value: reply})

	log.Debug("Reply : ", reply)

	log.Debug("[PipecouplerActivity:Eval] Exit ........ ")

	return true, nil
}

func (a *PipecouplerActivity) getPipeline(context activity.Context) (PipeCouplerClient, error) {
	log.Info("[PipecouplerActivity:getPipeline] entering ...... ")
	myId := util.ActivityId(context)
	client := a.clients[a.activityToModel[myId]]

	if nil == client {
		a.mux.Lock()
		defer a.mux.Unlock()
		client = a.clients[a.activityToModel[myId]]
		if nil == client {

			downstreamHosts, exist := context.GetSetting(DownstreamHost)
			if !exist {
				return nil, activity.NewError("Server is not configured", "Pipecoupler-4002", nil)
			}

			port, exist := context.GetSetting(Port)
			if !exist {
				return nil, activity.NewError("Server is not configured", "Pipecoupler-4002", nil)
			}

			log.Debug("[PipecouplerActivity:getPipeline] downstreamHost = ", downstreamHosts)
			var rootObject interface{}
			err := json.Unmarshal([]byte(downstreamHosts.(string)), &rootObject)
			if err != nil {
				return nil, err
			}
			downstreamHost := rootObject.([]interface{})[0].(string)

			address := fmt.Sprintf("%s:%d", downstreamHost, port)
			log.Info("[PipecouplerActivity:getPipeline] address = ", address)
			conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock()) //grpc.WithTimeout(time.Duration(5)*time.Second))
			if err != nil {
				log.Errorf("[PipecouplerActivity:getPipeline] Unable to connect: %v", err)
				//return nil, err
			}

			client = NewPipeCouplerClient(conn)
			a.clients[a.activityToModel[myId]] = client

		}
	}
	log.Info("[PipecouplerActivity:getPipeline] exit ...... ")

	return client, nil
}

func (a *PipecouplerActivity) Close() {
	//conn.Close()
}
