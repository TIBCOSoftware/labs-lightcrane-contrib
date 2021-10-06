/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package jsondeserializer

import (
	"encoding/json"
	"strings"

	"github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/util"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/TIBCOSoftware/flogo-lib/logger"
)

// activityLogger is the default logger for the Filter Activity
var log = logger.GetLogger("labs-lc-activity-jsondeserializer")

const (
	setting_DateFormat = "DateFormat"
	input              = "JSONString"
	output             = "Data"
)

// JSONDeserializerActivity is an Activity that is used to Filter a message to the console
type JSONDeserializerActivity struct {
	metadata *activity.Metadata
}

// NewActivity creates a new AppActivity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	aJSONDeserializerActivity := &JSONDeserializerActivity{
		metadata: metadata,
	}
	return aJSONDeserializerActivity
}

// Metadata returns the activity's metadata
func (a *JSONDeserializerActivity) Metadata() *activity.Metadata {
	return a.metadata
}

// Eval implements api.Activity.Eval - Filters the Message
func (a *JSONDeserializerActivity) Eval(ctx activity.Context) (done bool, err error) {

	logger.Info("[JSONDeserializerActivity:Eval] entering ........ ")
	defer logger.Info("[JSONDeserializerActivity:Eval] exit ........ ")

	in := strings.TrimSpace(ctx.GetInput(input).(string))
	var result interface{}
	if strings.HasPrefix(in, "{") {
		var rootObject interface{}
		err = json.Unmarshal([]byte(in), &rootObject)
		if nil != err {
			logger.Warn("[JSONDeserializerActivity:Eval] Unable to parse json data as object, reason : ", err.Error())
			return false, nil
		}
		rootMap, ok := rootObject.(map[string]interface{})
		if !ok {
			logger.Warn("[JSONDeserializerActivity:Eval] Unable to parse json data, reason : root object should be a map[string]interface{}")
			return false, nil
		}
		a.validate(ctx, rootMap)
		result = rootObject
	} else if strings.HasPrefix(in, "[") {
		var rootArray []interface{}
		err = json.Unmarshal([]byte(in), &rootArray)
		if nil != err {
			logger.Warn("[JSONDeserializerActivity:Eval] Unable to parse json data as array, reason : ", err.Error())
			return false, nil
		}
		result = rootArray
	}

	//	if nil == result {
	logger.Warn("[JSONDeserializerActivity:Eval] Unable to parse json data, reason : root object is nil")
	//		return false, nil
	//	}

	jsondata := &data.ComplexObject{Metadata: "Data", Value: result}

	ctx.SetOutput(output, jsondata)

	return true, nil
}

func (a *JSONDeserializerActivity) validate(ctx activity.Context, rootMap map[string]interface{}) {
	myId := util.ActivityId(ctx)
	defaultValues, ok := ctx.GetSetting("defaultValue")
	if !ok || nil == defaultValues {
		log.Warn("[JSONDeserializerActivity:Eval] No default values set!!")
		defaultValues = make([]interface{}, 0)
	}

	for _, defaultValue := range defaultValues.([]interface{}) {
		defaultValueMap := defaultValue.(map[string]interface{})
		log.Debug("[JSONDeserializerActivity:Eval] myId = ", myId, ", AttributePath = ", defaultValueMap["AttributePath"], ", Type = ", defaultValueMap["Type"], ", Default = ", defaultValueMap["Default"])
		attributePathElements := strings.Split(defaultValueMap["AttributePath"].(string), ".")
		currentMap := rootMap

		log.Debug("[JSONDeserializerActivity:Eval] rootMap[] = ", rootMap)
		for index, attributePathElement := range attributePathElements {
			if index == (len(attributePathElements) - 1) {
				/* the last element (attribute key) */
				if nil == currentMap[attributePathElement] {
					/* not exist then set to default */
					currentMap[attributePathElement] = defaultValueMap["Default"]
				}
				log.Debug("[JSONDeserializerActivity:Eval] currentMap[", attributePathElement, "] = ", currentMap[attributePathElement])
			} else {
				/* is a node not a leaf */
				if nil == currentMap[attributePathElement] {
					/* submap is not exist then create a new one */
					currentMap[attributePathElement] = make(map[string]interface{})
				}
				currentMap = currentMap[attributePathElement].(map[string]interface{})
				log.Debug("[JSONDeserializerActivity:Eval] currentMap[", attributePathElement, "] = ", currentMap)
			}
		}
	}
}
