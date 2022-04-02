/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */

package modelparameterbuilder

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	kwr "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/keywordreplace"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/objectbuilder"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

var log = logger.GetLogger("tibco-labs-modelparameterbuilder")

var initialized bool = false

const (
	sTemplateFolder     = "TemplateFolder"
	sLeftToken          = "leftToken"
	sRightToken         = "rightToken"
	sVariablesDef       = "variablesDef"
	sProperties         = "Properties"
	iFlogoAppDescriptor = "FlogoAppDescriptor"
	iRunner             = "Runner"
	iExtra              = "extra"
	iPorts              = "ports"
	iProperties         = "properties"
	iPropertyPrefix     = "PropertyPrefix"
	iServiceType        = "ServiceType"
	iVariable           = "Variables"
	oF1Properties       = "F1Properties"
	oDescriptor         = "Descriptor"
	oPropertyNameDef    = "PropertyNameDef"
)

type ModelParameterBuilderActivity struct {
	metadata    *activity.Metadata
	mux         sync.Mutex
	pathMappers map[string]*kwr.KeywordMapper
	variables   map[string]map[string]string
	gProperties map[string][]map[string]interface{}
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aModelParameterBuilderActivity := &ModelParameterBuilderActivity{
		metadata:    metadata,
		pathMappers: make(map[string]*kwr.KeywordMapper),
		variables:   make(map[string]map[string]string),
		gProperties: make(map[string][]map[string]interface{}),
	}

	return aModelParameterBuilderActivity
}

func (a *ModelParameterBuilderActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *ModelParameterBuilderActivity) Eval(context activity.Context) (done bool, err error) {

	log.Debug("[ModelParameterBuilderActivity:Eval] entering ........ ")
	defer log.Debug("[ModelParameterBuilderActivity:Eval] Exit ........ ")

	serviceType, ok := context.GetInput(iServiceType).(string)
	if !ok {
		return false, errors.New("Invalid Service Type ... ")
	}
	log.Debug("[ModelParameterBuilderActivity:Eval]  Name : ", serviceType)

	flogoAppDescriptor, ok := context.GetInput(iFlogoAppDescriptor).(map[string]interface{})
	if !ok {
		return false, errors.New("Invalid Flogo Application Descriptor ... ")
	}
	log.Debug("[ModelParameterBuilderActivity:Eval]  Flogo Application Descriptor : ", flogoAppDescriptor)

	/*********************************
	        Construct Pipeline
	**********************************/

	var gProperties []map[string]interface{}
	var runner string
	var ports []interface{}
	var appProperties []interface{}
	var extraArray []interface{}

	/* If runner defined */
	if nil != flogoAppDescriptor[iRunner] {
		runner = flogoAppDescriptor[iRunner].(string)
	}
	log.Debug("[ParameterBuilderActivity:Eval]  Runner : ", runner)

	if "docker-compose.yml" != runner {
		gProperties, err = a.getProperties(context)
		if err != nil {
			gProperties = nil
		}
	}

	if nil == gProperties {
		gProperties = make([]map[string]interface{}, 0)
	}

	/* If any server port defined */
	if nil != flogoAppDescriptor[iPorts] {
		ports = flogoAppDescriptor[iPorts].([]interface{})
	}

	/* Extrace Daynamic Parameter From DataSource */
	if nil != flogoAppDescriptor[iProperties] {
		appProperties = flogoAppDescriptor[iProperties].([]interface{})
	} else {
		appProperties = make([]interface{}, 0)
	}

	a.populateDefaultProperty(appProperties, "Working_Folder", "/app/artifacts")
	a.populateDefaultProperty(appProperties, "PythonModel_plugin", "artifacts.inference")
	a.populateDefaultProperty(appProperties, "System_ID", "$ID$")
	a.populateDefaultProperty(appProperties, "System_ServiceLocator", "$ServiceLocator$")
	a.populateDefaultProperty(appProperties, "System_EndpointComponent", "$ID$")
	a.populateDefaultProperty(appProperties, "System_Port", "10100")
	a.populateDefaultProperty(appProperties, "System_Standalone", "True")
	a.populateDefaultProperty(appProperties, "System_EchoOn", "True")

	if nil != flogoAppDescriptor[iExtra] {
		extraArray = flogoAppDescriptor[iExtra].([]interface{})
		for _, property := range extraArray {
			name := util.GetPropertyElement("Name", property).(string)
			if !strings.HasPrefix(name, "App.") {
				gProperties = append(gProperties, map[string]interface{}{
					"Name":  name,
					"Value": util.GetPropertyElement("Value", property),
					"Type":  util.GetPropertyElement("Type", property),
				})
			}
		}
	} else {
		extraArray = make([]interface{}, 0)
	}

	ports = a.populateDefaultPort(ports, appProperties)

	/*********************************
	    Construct Dynamic Parameter
	**********************************/

	propertyNameDef := map[string]interface{}{
		"Global": map[string]interface{}{},
	}
	gPropertyNameDef := propertyNameDef["Global"].(map[string]interface{})
	for _, property := range appProperties {
		name := property.(map[string]interface{})["Name"].(string)
		gPropertyNameDef[name] = name
		property.(map[string]interface{})["Name"] = name //strings.ReplaceAll(name, ".", "_")
	}

	pathMapper, _, _ := a.getVariableMapper(context)
	defVariable := context.GetInput(iVariable).(map[string]interface{})
	propertyPrefix, ok := context.GetInput(iPropertyPrefix).(string)
	if !ok {
		propertyPrefix = ""
	} else {
		propertyPrefix = pathMapper.Replace(propertyPrefix, defVariable)
	}
	log.Debug("[ModelParameterBuilderActivity:Eval]  Property Prefix : ", propertyPrefix)

	var f1Properties interface{}
	switch serviceType {
	case "k8s":
		f1Properties, _ = objectbuilder.CreateK8sF1Properties(
			pathMapper,
			defVariable,
			propertyPrefix,
			appProperties,
			gProperties,
			ports,
		)
	default:
		f1Properties, _ = objectbuilder.CreateDockerF1Properties(
			pathMapper,
			defVariable,
			propertyPrefix,
			appProperties,
			gProperties,
			ports,
		)
	}

	context.SetOutput(oF1Properties, f1Properties)
	context.SetOutput(oPropertyNameDef, propertyNameDef)
	log.Debug("[ModelParameterBuilderActivity:Eval]PropertyNameDef = ", propertyNameDef)

	return true, nil
}

func (a *ModelParameterBuilderActivity) populateDefaultProperty(properties []interface{}, name string, value string) []interface{} {
	for _, property := range properties {
		propertyName := property.(map[string]interface{})["Name"].(string)
		if propertyName == name {
			return properties
		}
	}
	return append(properties, map[string]interface{}{
		"Name":  name,
		"Value": value,
	})
}

func (a *ModelParameterBuilderActivity) populateDefaultPort(ports []interface{}, properties []interface{}) []interface{} {
	if nil == ports {
		ports = make([]interface{}, 0)
	}
	port := ""
	extPort := ""
	endpointPort := ""
	for _, property := range properties {
		propertyName := property.(map[string]interface{})["Name"].(string)
		if "System_Port" == propertyName {
			port = property.(map[string]interface{})["Value"].(string)
		} else if "System_Port_Ext" == propertyName {
			extPort = property.(map[string]interface{})["Value"].(string)
		} else if "System_ExternalEndpointPort" == propertyName {
			endpointPort = property.(map[string]interface{})["Value"].(string)
		}
	}

	if "" == extPort {
		extPort = endpointPort
	}

	if "" == port || "" == extPort {
		return ports
	}

	defaultPortPair := fmt.Sprintf("%s:%s", extPort, port)
	for _, aPortPair := range ports {
		if strings.HasSuffix(aPortPair.(string), fmt.Sprintf(":%s", port)) {
			return ports
		}
	}

	return append(ports, defaultPortPair)
}

func (a *ModelParameterBuilderActivity) getProperties(ctx activity.Context) ([]map[string]interface{}, error) {

	log.Debug("[ModelParameterBuilderActivity:getTemplate] entering ........ ")
	defer log.Debug("[ModelParameterBuilderActivity:getTemplate] exit ........ ")

	myId := util.ActivityId(ctx)
	gProperties := a.gProperties[myId]

	if nil == gProperties {
		a.mux.Lock()
		defer a.mux.Unlock()
		gProperties = a.gProperties[myId]
		if nil == gProperties {
			gPropertiesSetting, exist := ctx.GetSetting(sProperties)
			gProperties = make([]map[string]interface{}, 0)
			if exist {
				for _, gProperty := range gPropertiesSetting.([]interface{}) {
					gProperties = append(gProperties, gProperty.(map[string]interface{}))
				}
			}
			a.gProperties[myId] = gProperties
		}
	}
	return gProperties, nil
}

func (a *ModelParameterBuilderActivity) getVariableMapper(ctx activity.Context) (*kwr.KeywordMapper, map[string]string, error) {
	myId := util.ActivityId(ctx)
	mapper := a.pathMappers[myId]
	variables := a.variables[myId]

	if nil == mapper {
		a.mux.Lock()
		defer a.mux.Unlock()
		mapper = a.pathMappers[myId]
		if nil == mapper {
			variables = make(map[string]string)
			variablesDef, ok := ctx.GetSetting(sVariablesDef)
			log.Debug("[ModelParameterBuilderActivity:getTemplate] Processing handlers : variablesDef = ", variablesDef)
			if ok && nil != variablesDef {
				for _, variableDef := range variablesDef.([]interface{}) {
					variableInfo := variableDef.(map[string]interface{})
					variables[variableInfo["Name"].(string)] = variableInfo["Type"].(string)
				}
			}

			lefttoken, exist := ctx.GetSetting(sLeftToken)
			if !exist {
				return nil, nil, errors.New("LeftToken not defined!")
			}
			righttoken, exist := ctx.GetSetting(sRightToken)
			if !exist {
				return nil, nil, errors.New("RightToken not defined!")
			}
			mapper = kwr.NewKeywordMapper("", lefttoken.(string), righttoken.(string))

			a.pathMappers[myId] = mapper
			a.variables[myId] = variables
		}
	}
	return mapper, variables, nil
}
