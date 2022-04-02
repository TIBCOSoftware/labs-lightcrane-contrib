/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */

/*
	{
		"imports": [],
		"name": "ProjectAirApplication",
		"description": "",
		"version": "1.0.0",
		"type": "flogo:app",
		"appModel": "1.1.1",
		"feVersion": "2.9.0",
		"triggers": [],
		"resources": [],
		"properties": [],
		"connections": {},
		"contrib": "",
		"fe_metadata": ""
	}
*/

package airparameterbuilder

import (
	"errors"
	"strings"
	"sync"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	kwr "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/keywordreplace"

	model "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/airmodel"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/objectbuilder"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

var log = logger.GetLogger("tibco-model-ops-pipelinebuilder")

var initialized bool = false

const (
	sTemplateFolder             = "TemplateFolder"
	sLeftToken                  = "leftToken"
	sRightToken                 = "rightToken"
	sVariablesDef               = "variablesDef"
	sProperties                 = "Properties"
	iApplicationName            = "ApplicationName"
	iApplicationProperties      = "ApplicationProperties"
	iFlogoAppDescriptor         = "FlogoAppDescriptor"
	iRunner                     = "Runner"
	iExtra                      = "extra"
	iPorts                      = "ports"
	iProperties                 = "properties"
	iPropertyPrefix             = "PropertyPrefix"
	iServiceType                = "ServiceType"
	iVariable                   = "Variables"
	oFlogoApplicationDescriptor = "FlogoDescriptor"
	oF1Properties               = "F1Properties"
	oDescriptor                 = "Descriptor"
	oPropertyNameDef            = "PropertyNameDef"
)

type ParameterBuilderActivity struct {
	metadata    *activity.Metadata
	mux         sync.Mutex
	templates   map[string]*model.FlogoTemplateLibrary
	pathMappers map[string]*kwr.KeywordMapper
	variables   map[string]map[string]string
	gProperties map[string][]map[string]interface{}
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aParameterBuilderActivity := &ParameterBuilderActivity{
		metadata:    metadata,
		templates:   make(map[string]*model.FlogoTemplateLibrary),
		pathMappers: make(map[string]*kwr.KeywordMapper),
		variables:   make(map[string]map[string]string),
		gProperties: make(map[string][]map[string]interface{}),
	}

	return aParameterBuilderActivity
}

func (a *ParameterBuilderActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *ParameterBuilderActivity) Eval(context activity.Context) (done bool, err error) {

	log.Debug("[ParameterBuilderActivity:Eval] entering ........ ")
	defer log.Debug("[ParameterBuilderActivity:Eval] Exit ........ ")

	_, gProperties, err := a.getTemplateLibrary(context)
	if err != nil {
		return false, err
	}

	serviceType, ok := context.GetInput(iServiceType).(string)
	if !ok {
		return false, errors.New("Invalid Service Type ... ")
	}
	log.Debug("[ParameterBuilderActivity:Eval]  Name : ", serviceType)

	flogoAppDescriptor, ok := context.GetInput(iFlogoAppDescriptor).(map[string]interface{})
	if !ok {
		return false, errors.New("Invalid Flogo Application Descriptor ... ")
	}
	log.Debug("[ParameterBuilderActivity:Eval]  Flogo Application Descriptor : ", flogoAppDescriptor)

	/*********************************
	        Construct Pipeline
	**********************************/

	var runner string
	var ports []interface{}
	var appProperties []interface{}
	var extraArray []interface{}

	/* If runner defined */
	if nil != flogoAppDescriptor[iRunner] {
		runner = flogoAppDescriptor[iRunner].(string)
	}
	log.Debug("[ParameterBuilderActivity:Eval]  Runner : ", runner)

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
	//	appProperties = append(appProperties, map[string]interface{}{
	//		"Name":  "FLOGO_APP_PROPS_ENV",
	//		"Value": "auto",
	//	})

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
		property.(map[string]interface{})["Name"] = strings.ReplaceAll(name, ".", "_")
	}

	pathMapper, _, _ := a.getVariableMapper(context)
	defVariable := context.GetInput(iVariable).(map[string]interface{})
	propertyPrefix, ok := context.GetInput(iPropertyPrefix).(string)
	if !ok {
		propertyPrefix = ""
	} else {
		propertyPrefix = pathMapper.Replace(propertyPrefix, defVariable)
	}
	log.Debug("[ParameterBuilderActivity:Eval]  Property Prefix : ", propertyPrefix)

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
	log.Debug("[PipelineBuilderActivity:Eval]PropertyNameDef = ", propertyNameDef)

	return true, nil
}

func (a *ParameterBuilderActivity) getTemplateLibrary(ctx activity.Context) (*model.FlogoTemplateLibrary, []map[string]interface{}, error) {

	log.Debug("[ParameterBuilderActivity:getTemplate] entering ........ ")
	defer log.Debug("[ParameterBuilderActivity:getTemplate] exit ........ ")

	myId := util.ActivityId(ctx)
	templateLib := a.templates[myId]
	gProperties := a.gProperties[myId]

	if nil == templateLib {
		a.mux.Lock()
		defer a.mux.Unlock()
		templateLib = a.templates[myId]
		gProperties = a.gProperties[myId]
		if nil == templateLib {
			templateFolderSetting, exist := ctx.GetSetting(sTemplateFolder)
			if !exist {
				return nil, nil, activity.NewError("Template is not configured", "PipelineBuilder-4002", nil)
			}
			templateFolder := templateFolderSetting.(string)
			var err error
			templateLib, err = model.NewFlogoTemplateLibrary(templateFolder)
			if nil != err {
				return nil, nil, err
			}

			a.templates[myId] = templateLib
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
	return templateLib, gProperties, nil
}

func (a *ParameterBuilderActivity) getVariableMapper(ctx activity.Context) (*kwr.KeywordMapper, map[string]string, error) {
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
			log.Debug("Processing handlers : variablesDef = ", variablesDef)
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
