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
	"fmt"
	"strings"
	"sync"

	kwr "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/keywordreplace"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"

	//	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/objectbuilder"
	model "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/airmodel"
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

	var ports []interface{}
	var appProperties []interface{}
	var extraArray []interface{}

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
		f1Properties, _ = a.createK8sF1Properties(
			pathMapper,
			defVariable,
			propertyPrefix,
			appProperties,
			gProperties,
			ports,
		)
	default:
		f1Properties, _ = a.createDockerF1Properties(
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

func (a *ParameterBuilderActivity) createDockerF1Properties(
	pathMapper *kwr.KeywordMapper,
	defVariable map[string]interface{},
	propertyPrefix string,
	appProperties []interface{},
	gProperties []map[string]interface{},
	ports []interface{},
) (interface{}, error) {

	description := make([]interface{}, 0)
	mainDescription := map[string]interface{}{
		"Group": "main",
		"Value": make([]interface{}, 0),
	}
	description = append(description, mainDescription)

	for _, property := range gProperties {
		/* nil will bot be accepted */
		value, dtype, err := util.GetPropertyValue(property["Value"], property["Type"])
		if nil != err {
			return nil, err
		}
		log.Debug("[createDockerF1Properties] Name = ", property["Name"], ", Raw Value = ", value, ", defVariable = ", defVariable)

		if "String" == dtype {
			value = pathMapper.Replace(value.(string), defVariable)
			log.Debug("[createDockerF1Properties] Value after replace = ", value)
			sValue := value.(string)
			if "" != sValue && sValue[0] == '$' && sValue[len(sValue)-1] == '$' {
				continue
			}
		}
		mainDescription["Value"] = append(mainDescription["Value"].([]interface{}), map[string]interface{}{
			"Name":  pathMapper.Replace(util.GetPropertyElementAsString("Name", property), defVariable),
			"Value": value,
			"Type":  util.GetPropertyElementAsString("Type", property),
		})
	}
	for index, property := range appProperties {
		mainDescription["Value"] = append(mainDescription["Value"].([]interface{}), map[string]interface{}{
			"Name":  pathMapper.Replace(fmt.Sprintf("%s.environment[%d]", propertyPrefix, index), defVariable),
			"Value": fmt.Sprintf("%s=%s", util.GetPropertyElement("Name", property), util.GetPropertyElement("Value", property)),
			"Type":  util.GetPropertyElement("Type", property),
		})
	}
	index := 0
	for _, port := range ports {
		mainDescription["Value"] = append(mainDescription["Value"].([]interface{}), map[string]interface{}{
			"Name":  pathMapper.Replace(fmt.Sprintf("%s.ports[%d]", propertyPrefix, index), defVariable),
			"Value": port,
			"Type":  "String",
		})
		index++
	}
	return description, nil
}

func (a *ParameterBuilderActivity) createK8sF1Properties(
	pathMapper *kwr.KeywordMapper,
	defVariable map[string]interface{},
	propertyPrefix string,
	appProperties []interface{},
	gProperties []map[string]interface{},
	ports []interface{},
) (interface{}, error) {
	groupProperties := make(map[string]interface{})
	for _, property := range gProperties {
		name := util.GetPropertyElementAsString("Name", property)
		group := name[0:strings.Index(name, "_")]
		if nil == groupProperties[group] {
			groupProperties[group] = make([]interface{}, 0)
		}
		name = name[strings.Index(name, "_")+1 : len(name)]
		property["Name"] = name
		groupProperties[group] = append(groupProperties[group].([]interface{}), property)
	}
	/*
		{
			"Group":"main",
			"Value":[
				{"Name":"apiVersion","Type":null,"Value":"apps/v1"},
				{"Name":"kind","Type":null,"Value":"Deployment"},
				{"Name":"metadata.name","Type":null,"Value":"http_dummy"},
				{"Name":"spec.template.spec.containers[0].image","Type":null,"Value":"bigoyang/http_dummy:0.2.1"},
				{"Name":"spec.template.spec.containers[0].name","Type":null,"Value":"http_dummy"},
				{"Name":"spec.selector.matchLabels.component","Type":null,"Value":"http_dummy"},
				{"Name":"spec.template.metadata.labels.component","Type":null,"Value":"http_dummy"},
				{"Name":"spec.template.spec.containers[0].env[0].name","Type":"string","Value":"Logging_LogLevel"},
				{"Name":"spec.template.spec.containers[0].env[0].value","Type":null,"Value":"INFO"},
				{"Name":"spec.template.spec.containers[0].env[1].name","Type":"string","Value":"FLOGO_APP_PROPS_ENV"},
				{"Name":"spec.template.spec.containers[0].env[1].value","Type":null,"Value":"auto"},
				{"Name":"spec.template.spec.containers[0].ports[0]","Type":"String","Value":"9999"}
			]
		},
	*/
	description := make([]interface{}, 0)
	mainDescription := map[string]interface{}{
		"Group": "main",
		"Value": make([]interface{}, 0),
	}
	description = append(description, mainDescription)

	for _, iProperty := range groupProperties["main"].([]interface{}) {
		property := iProperty.(map[string]interface{})
		value, dtype, err := util.GetPropertyValue(property["Value"], property["Type"])
		if nil != err {
			return nil, err
		}
		if "String" == dtype {
			value = pathMapper.Replace(value.(string), defVariable)
		}
		mainDescription["Value"] = append(mainDescription["Value"].([]interface{}), map[string]interface{}{
			"Name":  pathMapper.Replace(util.GetPropertyElementAsString("Name", property), defVariable),
			"Value": value,
			"Type":  util.GetPropertyElement("Type", property),
		})
	}
	for index, property := range appProperties {
		mainDescription["Value"] = append(mainDescription["Value"].([]interface{}), map[string]interface{}{
			"Name":  pathMapper.Replace(fmt.Sprintf("%s.env[%d].name", propertyPrefix, index), defVariable),
			"Value": util.GetPropertyElement("Name", property),
			"Type":  "string",
		})
		mainDescription["Value"] = append(mainDescription["Value"].([]interface{}), map[string]interface{}{
			"Name":  pathMapper.Replace(fmt.Sprintf("%s.env[%d].value", propertyPrefix, index), defVariable),
			"Value": util.GetPropertyElement("Value", property),
			"Type":  util.GetPropertyElement("Type", property),
		})
	}

	if nil != ports && 0 < len(ports) {
		ipServiceDescription := map[string]interface{}{
			"Group": "ip-service",
			"Value": make([]interface{}, 0),
		}
		description = append(description, ipServiceDescription)

		/*
			{
				"Group":"ip-service",
				"Value":[
					{"Name":"apiVersion","Type":"String","Value":"v1"},
					{"Name":"kind","Type":"String","Value":"Service"},
					{"Name":"metadata.name","Type":"String","Value":"$name$-ip-service"},
					{"Name":"spec.selector.component","Type":"String","Value":"$name$"},
					{"Name":"spec.type","Type":"String","Value":"LoadBalancer"},
					{"Name":"spec.port[0]","Type":"String","Value":"8080"},
					{"Name":"spec.targetPort[0]","Type":"String","Value":"9999"}
				]
			}
		*/
		for _, iProperty := range groupProperties["ip-service"].([]interface{}) {
			property := iProperty.(map[string]interface{})
			value, dtype, err := util.GetPropertyValue(property["Value"], property["Type"])
			if nil != err {
				return nil, err
			}
			if "String" == dtype {
				value = pathMapper.Replace(value.(string), defVariable)
			}
			ipServiceDescription["Value"] = append(ipServiceDescription["Value"].([]interface{}), map[string]interface{}{
				"Name":  pathMapper.Replace(util.GetPropertyElementAsString("Name", property), defVariable),
				"Value": value,
				"Type":  util.GetPropertyElement("Type", property),
			})
		}

		index := 0
		for _, port := range ports {
			portPair := strings.Split(port.(string), ":")
			mainDescription["Value"] = append(mainDescription["Value"].([]interface{}), map[string]interface{}{
				"Name":  pathMapper.Replace(fmt.Sprintf("%s.ports[%d]", propertyPrefix, index), defVariable),
				"Value": portPair[1],
				"Type":  "String",
			})

			ipServiceDescription["Value"] = append(ipServiceDescription["Value"].([]interface{}), map[string]interface{}{
				"Name":  fmt.Sprintf("spec.ports[%d].port", index),
				"Value": portPair[0],
				"Type":  "String",
			})
			ipServiceDescription["Value"] = append(ipServiceDescription["Value"].([]interface{}), map[string]interface{}{
				"Name":  fmt.Sprintf("spec.ports[%d].targetPort", index),
				"Value": portPair[1],
				"Type":  "String",
			})
			index++
		}
	}

	return description, nil
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
