/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package properties2object

import (
	"errors"
	"strconv"
	"strings"
	"sync"

	kwr "github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/keywordreplace"
	"github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/util"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
)

var log = logger.GetLogger("tibco-model-ops-cmdconverter")

var initialized bool = false

const (
	sLeftToken          = "leftToken"
	sRightToken         = "rightToken"
	iVariable           = "Variables"
	iProperties         = "Properties"
	iOverrideProperties = "OverrideProperties"
	iPassThroughData    = "PassThroughData"
	oPassThroughDataOut = "PassThroughDataOut"
	oDataObject         = "DataObject"
)

type Properties2ObjectActivity struct {
	metadata        *activity.Metadata
	mux             sync.Mutex
	variableMappers map[string]*kwr.KeywordMapper
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aProperties2ObjectActivity := &Properties2ObjectActivity{
		metadata:        metadata,
		variableMappers: make(map[string]*kwr.KeywordMapper),
	}

	return aProperties2ObjectActivity
}

func (a *Properties2ObjectActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *Properties2ObjectActivity) Eval(context activity.Context) (done bool, err error) {

	log.Info("[Properties2ObjectActivity:Eval] entering ........ ")

	propertiesGroups, ok := context.GetInput(iProperties).([]interface{})
	if !ok {
		return true, errors.New("Illegal input propertiesGroup!!")
	}

	log.Info("[Properties2ObjectActivity:Eval] ??????????????????? ", propertiesGroups)

	overrideProperties, _ := context.GetInput(iOverrideProperties).(map[string]interface{})

	variable := a.getVariables(context)

	variableMapper, _ := a.getVariableMapper(context)

	propertiesGroupsObj := make(map[string]interface{})
	for _, propertiesGroup := range propertiesGroups {
		log.Info("[Properties2ObjectActivity:Eval] ??????????????????? ", propertiesGroup)
		propertiesGroupObj, err := a.buildObject(
			propertiesGroup.(map[string]interface{})["Value"].([]interface{}),
			variableMapper,
			variable,
			overrideProperties,
		)
		if nil != err {
			return true, err
		}

		propertiesGroupsObj[propertiesGroup.(map[string]interface{})["Group"].(string)] = propertiesGroupObj
	}

	context.SetOutput(oDataObject, propertiesGroupsObj)
	if nil != context.GetInput(iPassThroughData) {
		context.SetOutput(oPassThroughDataOut, context.GetInput(iPassThroughData))
	}
	log.Info("[Properties2ObjectActivity:Eval] oDataObject : ", propertiesGroupsObj)

	log.Info("[Properties2ObjectActivity:Eval] Exit ........ ")

	return true, nil
}

func (a *Properties2ObjectActivity) buildObject(
	properties []interface{},
	variableMapper *kwr.KeywordMapper,
	variable map[string]interface{},
	overrideProperties map[string]interface{},
) (map[string]interface{}, error) {

	propertiesObj := make(map[string]interface{})
	for _, iProperty := range properties {
		property := iProperty.(map[string]interface{})
		propertyKey := variableMapper.Replace(property["Name"].(string), variable)
		var keyElements []string
		if strings.Contains(propertyKey, "..") {
			pos := strings.Index(propertyKey, "..")
			keyElements = strings.Split(propertyKey[:pos], ".")
			keyElements = append(keyElements, propertyKey[pos+1:])
		} else {
			keyElements = strings.Split(propertyKey, ".")
		}

		log.Info("[Properties2ObjectActivity:Eval] keyElements = ", keyElements)

		if nil != property["Value"] {
			propertyValue, trueType, err := util.GetPropertyValue(property["Value"], property["Type"])
			if nil != err {
				return nil, err
			}
			if "String" == trueType {
				propertyValue = variableMapper.Replace(propertyValue.(string), variable)
			}
			log.Info("(Properties2ObjectActivity.Eval) propertyValue : ", propertyValue)
			current := propertiesObj
			for index, key := range keyElements {
				//log.Info("[Properties2ObjectActivity:Eval] key = ", key)
				if strings.HasSuffix(key, "]") {
					//log.Info("[Properties2ObjectActivity:Eval] an array ... ")
					pos := strings.Index(key, "[")
					slot, _ := strconv.Atoi(key[pos+1 : len(key)-1])
					key = key[0:pos]
					if nil == current[key] {
						current[key] = make([]interface{}, 0)
					}
					if index == len(keyElements)-1 {
						/* It's an primitive array element.*/
						current[key] = append(current[key].([]interface{}), propertyValue)
					} else {
						for len(current[key].([]interface{}))-1 < slot {
							current[key] = append(current[key].([]interface{}), make(map[string]interface{}))
						}
						current = current[key].([]interface{})[slot].(map[string]interface{})
					}
				} else {
					//log.Info("[Properties2ObjectActivity:Eval] an object or a value ... ")
					if nil == current[key] {
						current[key] = make(map[string]interface{})
					}
					if index == len(keyElements)-1 {
						/* It's an object attribute.*/
						current[key] = propertyValue
					} else {
						current = current[key].(map[string]interface{})
					}
				}
			}
		}
	}

	log.Info("propertiesObj : ", propertiesObj)

	return propertiesObj, nil
}

func (a *Properties2ObjectActivity) getVariables(context activity.Context) map[string]interface{} {

	temp := make(map[string]interface{})
	variables, ok := context.GetInput(iVariable).(map[string]interface{})
	if !ok {
		return temp
	}
	for key, value := range variables {
		switch value.(type) {
		case string:
			temp[key] = value
		default:
			continue
		}
	}
	return temp
}

func (a *Properties2ObjectActivity) getVariableMapper(ctx activity.Context) (*kwr.KeywordMapper, error) {
	myId := util.ActivityId(ctx)
	mapper := a.variableMappers[myId]

	if nil == mapper {
		a.mux.Lock()
		defer a.mux.Unlock()
		mapper = a.variableMappers[myId]
		if nil == mapper {
			lefttoken, exist := ctx.GetSetting(sLeftToken)
			if !exist {
				return nil, errors.New("LeftToken not defined!")
			}
			righttoken, exist := ctx.GetSetting(sRightToken)
			if !exist {
				return nil, errors.New("RightToken not defined!")
			}
			mapper = kwr.NewKeywordMapper("", lefttoken.(string), righttoken.(string))

			a.variableMappers[myId] = mapper
		}
	}
	return mapper, nil
}
