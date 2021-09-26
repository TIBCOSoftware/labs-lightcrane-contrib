/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package objectserializer

import (
	"encoding/json"
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
)

var log = logger.GetLogger("activity-objectserializer")

const (
	sStringFormat       = "StringFormat"
	iData               = "Data"
	iPassThroughData    = "PassThroughData"
	oPassThroughDataOut = "PassThroughDataOut"
	oSerializedString   = "SerializedString"
	fJson               = "json"
	fYaml               = "yaml"
	fSimple             = "simple"
)

type ObjectSerializerActivity struct {
	metadata *activity.Metadata
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aObjectSerializerActivity := &ObjectSerializerActivity{
		metadata: metadata,
	}
	return aObjectSerializerActivity
}

func (a *ObjectSerializerActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *ObjectSerializerActivity) Eval(ctx activity.Context) (done bool, err error) {

	dataGroups, ok := ctx.GetInput(iData).(map[string]interface{})
	if !ok {
		log.Warn("No valid data ... ")
	}
	fmt.Println(">>>>>>>>>1", dataGroups)

	format, _ := ctx.GetSetting(sStringFormat)
	var serializedData interface{}
	switch format {
	case fJson:
		serializedData, err = buildJSON(dataGroups)
	case fYaml:
		serializedData, err = buildYAML(dataGroups)
	default:
		serializedData = fmt.Sprintf("%s", dataGroups)
	}

	if nil != err {
		return true, err
	}

	ctx.SetOutput(oSerializedString, serializedData)
	if nil != ctx.GetInput(iPassThroughData) {
		ctx.SetOutput(oPassThroughDataOut, ctx.GetInput(iPassThroughData))
	}

	return true, nil
}

func buildJSON(data map[string]interface{}) (interface{}, error) {
	var jsondata []byte
	var err error
	if 1 < len(data) {
		jsondata, err = json.Marshal(data)
	} else {
		/* Throw away envelope */
		for _, element := range data {
			jsondata, err = json.Marshal(element)
		}
	}
	if nil != err {
		return nil, err
	}
	return string(jsondata), nil
}

func buildYAML(dataGroups map[string]interface{}) (interface{}, error) {
	fmt.Println(">>>>>>>>>2", dataGroups)

	yamlString := ""
	index := 0
	for _, data := range dataGroups {
		jamldata, err := yaml.Marshal(data)
		if nil != err {
			return nil, err
		}
		if 0 < index {
			yamlString += "---\n"
		}
		yamlString += string(jamldata)
		index++
	}

	fmt.Println(">>>>>>>>>3", yamlString)

	return yamlString, nil
}
