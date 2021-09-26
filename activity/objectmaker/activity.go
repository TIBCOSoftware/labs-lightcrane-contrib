/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package objectmaker

import (
	"strings"

	"github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/util"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
)

// activityLogger is the default logger for the Filter Activity
var log = logger.GetLogger("activity-objectmaker")

const (
	setting_DateFormat = "DateFormat"
	iObjectDataMapping = "ObjectDataMapping"
	oObjectOut         = "ObjectOut"
)

// ObjectMakerActivity is an Activity that is used to Filter a message to the console
type ObjectMakerActivity struct {
	metadata *activity.Metadata
}

// NewActivity creates a new AppActivity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	aObjectMakerActivity := &ObjectMakerActivity{
		metadata: metadata,
	}
	return aObjectMakerActivity
}

// Metadata returns the activity's metadata
func (a *ObjectMakerActivity) Metadata() *activity.Metadata {
	return a.metadata
}

// Eval implements api.Activity.Eval - Filters the Message
func (a *ObjectMakerActivity) Eval(ctx activity.Context) (done bool, err error) {

	data := ctx.GetInput(iObjectDataMapping)

	var result interface{}
	switch data.(type) {
	case map[string]interface{}:
		newObject := data.(map[string]interface{})
		a.validate(ctx, newObject)
		result = newObject
	case []interface{}:
		result = data.([]interface{})
	default:
		logger.Warn("Unable to build object, reason : not a valid object")
		return false, nil
	}

	ctx.SetOutput(oObjectOut, result)

	return true, nil
}

func (a *ObjectMakerActivity) validate(ctx activity.Context, rootMap map[string]interface{}) {
	myId := util.ActivityId(ctx)
	defaultValues, ok := ctx.GetSetting("defaultValue")
	if !ok || nil == defaultValues {
		log.Info("No default values set!!")
	} else {
		for _, defaultValue := range defaultValues.([]interface{}) {
			defaultValueMap := defaultValue.(map[string]interface{})
			log.Debug("myId = ", myId, ", AttributePath = ", defaultValueMap["AttributePath"], ", Type = ", defaultValueMap["Type"], ", Default = ", defaultValueMap["Default"])
			attributePathElements := strings.Split(defaultValueMap["AttributePath"].(string), ".")
			currentMap := rootMap

			log.Debug("rootMap[] = ", rootMap)
			for index, attributePathElement := range attributePathElements {
				if index == (len(attributePathElements) - 1) {
					/* the last element (attribute key) */
					if nil == currentMap[attributePathElement] {
						/* not exist then set to default */
						currentMap[attributePathElement] = defaultValueMap["Default"]
					}
					log.Debug("currentMap[", attributePathElement, "] = ", currentMap[attributePathElement])
				} else {
					/* is a node not a leaf */
					if nil == currentMap[attributePathElement] {
						/* submap is not exist then create a new one */
						currentMap[attributePathElement] = make(map[string]interface{})
					}
					currentMap = currentMap[attributePathElement].(map[string]interface{})
					log.Debug("currentMap[", attributePathElement, "] = ", currentMap)
				}
			}
		}
	}
}
