/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package pipecoupler

import (
	ctx "context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"

	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/metadata"
)

const (
	DownstreamHost = "DownstreamHost"
	Port           = "Port"
	iTimeout       = "Timeout"
	iData          = "Data"
	oReply         = "Reply"
)

type Settings struct {
	DownstreamHost string `md:"DownstreamHost"`
	Port           int    `md:"Port"`
}

type Input struct {
	Timeout int                    `md:"Timeout"`
	Data    map[string]interface{} `md:"Data"`
}

type Output struct {
	Reply map[string]interface{} `md:"Reply"`
}

func (i *Input) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Timeout": i.Timeout,
		"Data":    i.Data,
	}
}

func (i *Input) FromMap(values map[string]interface{}) error {
	ok := true
	i.Timeout, ok = values["Timeout"].(int)
	if !ok {
		return errors.New("Illegal Timeout type, expect int.")
	}
	i.Data, ok = values["Data"].(map[string]interface{})
	if !ok {
		return errors.New("Illegal Data type, expect map[string]interface{}.")
	}
	return nil
}

func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Reply": o.Reply,
	}
}

func (o *Output) FromMap(values map[string]interface{}) error {
	ok := true
	o.Reply, ok = values["Reply"].(map[string]interface{})
	if !ok {
		return errors.New("Illegal Reply type, expect map[string]interface{}.")
	}
	return nil
}

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

func init() {
	_ = activity.Register(&Activity{}, New)
}

type Activity struct {
	activityToModel map[string]string
	client          PipeCouplerClient
}

func New(ctx activity.InitContext) (activity.Activity, error) {
	settings := &Settings{}
	err := metadata.MapToStruct(ctx.Settings(), settings, true)
	if err != nil {
		return nil, err
	}

	downstreamHosts := settings.DownstreamHost
	if "" == downstreamHosts {
		return nil, activity.NewError("Server is not configured", "Pipecoupler-4002", nil)
	}

	port := settings.Port
	if 0 > port {
		return nil, activity.NewError("Server is not configured", "Pipecoupler-4002", nil)
	}

	var rootObject interface{}
	err = json.Unmarshal([]byte(downstreamHosts), &rootObject)
	if err != nil {
		return nil, err
	}
	downstreamHost := rootObject.([]interface{})[0].(string)

	address := fmt.Sprintf("%s:%d", downstreamHost, port)
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock()) //grpc.WithTimeout(time.Duration(5)*time.Second))
	if err != nil {
		return nil, err
	}

	activity := &Activity{
		client: NewPipeCouplerClient(conn),
	}

	return activity, nil
}

func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

func (a *Activity) Eval(context activity.Context) (done bool, err error) {

	log := context.Logger()
	log.Debug("[Pipecoupler:Eval] entering ........ ")
	defer log.Debug("[Pipecoupler:Eval] exit ........ ")

	input := &Input{}
	context.GetInputObject(input)

	timeout := input.Timeout
	if 0 > timeout {
		timeout = 30
	}
	log.Debug("[PipecouplerActivity:Eval] timeout: ", timeout)

	dataMap := input.Data
	if nil == dataMap {
		return false, errors.New("No data comes in ... ")
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

	//reqContext, cancel := ctx.WithDeadline(ctx.Background(), time.Now().Add(time.Duration(timeout)*time.Second))
	reqContext, cancel := ctx.WithTimeout(ctx.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	replyObj, err := a.client.HandleData(
		reqContext,
		&Data{
			Sender:  sender,
			ID:      id,
			Content: content,
		},
	)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Error from down stream : %v", err))
	}

	reply := map[string]interface{}{
		"Sender":  replyObj.GetSender(),
		"ID":      replyObj.GetID(),
		"Content": replyObj.GetContent(),
		"Status":  replyObj.GetStatus(),
	}

	context.SetOutput(oReply, reply)

	log.Debug("Reply : ", reply)

	return true, nil
}
