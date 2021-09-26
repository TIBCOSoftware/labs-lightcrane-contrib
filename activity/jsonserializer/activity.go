/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package jsonserializer

import (
	"encoding/json"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/TIBCOSoftware/flogo-lib/logger"
)

var log = logger.GetLogger("labs-lc-activity-jsonserializer")

const (
	iData       = "Data"
	oJSONString = "JSONString"
)

type JSONSerializerActivity struct {
	metadata *activity.Metadata
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aJSONSerializerActivity := &JSONSerializerActivity{
		metadata: metadata,
	}
	return aJSONSerializerActivity
}

func (a *JSONSerializerActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *JSONSerializerActivity) Eval(ctx activity.Context) (done bool, err error) {

	log.Info("[JSONSerializerActivity:Eval] entering ........ ")
	defer log.Info("[JSONSerializerActivity:Eval] exit ........ ")

	data, ok := ctx.GetInput(iData).(*data.ComplexObject).Value.(map[string]interface{})

	log.Debug("[JSONSerializerActivity:Eval] data in : ", data)

	if !ok {
		log.Warn("[JSONSerializerActivity:Eval] No valid data ... ")
	}

	jsondata, err := json.Marshal(data)
	if nil != err {
		logger.Warn("[JSONSerializerActivity:Eval] Unable to serialize object, reason : ", err.Error())
		return false, nil
	}

	log.Debug("[JSONSerializerActivity:Eval] json out : ", string(jsondata))

	ctx.SetOutput(oJSONString, string(jsondata))

	return true, nil
}
