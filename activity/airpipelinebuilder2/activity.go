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

package airpipelinebuilder2

import (
	"encoding/json"
	"errors"

	"fmt"
	"strings"
	"sync"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	kwr "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/keywordreplace"

	//	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/objectbuilder"
	model "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/airmodel"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

var log = logger.GetLogger("tibco-lc-pipelinebuilder2")

var initialized bool = false

const (
	sTemplateFolder                = "TemplateFolder"
	sLeftToken                     = "leftToken"
	sRightToken                    = "rightToken"
	sVariablesDef                  = "variablesDef"
	sProperties                    = "Properties"
	iApplicationName               = "ApplicationName"
	iApplicationProperties         = "ApplicationProperties"
	iApplicationPipelineDescriptor = "AirDescriptor"
	iPorts                         = "ports"
	iProperties                    = "properties"
	iPropertyPrefix                = "PropertyPrefix"
	iServiceType                   = "ServiceType"
	iVariable                      = "Variables"
	oFlogoApplicationDescriptor    = "FlogoDescriptor"
	oF1Properties                  = "F1Properties"
	oDescriptor                    = "Descriptor"
	oPropertyNameDef               = "PropertyNameDef"
	oRunner                        = "Runner"
)

type PipelineBuilderActivity2 struct {
	metadata    *activity.Metadata
	mux         sync.Mutex
	templates   map[string]*model.FlogoTemplateLibrary
	pathMappers map[string]*kwr.KeywordMapper
	variables   map[string]map[string]string
	gProperties map[string][]map[string]interface{}
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aPipelineBuilderActivity2 := &PipelineBuilderActivity2{
		metadata:    metadata,
		templates:   make(map[string]*model.FlogoTemplateLibrary),
		pathMappers: make(map[string]*kwr.KeywordMapper),
		variables:   make(map[string]map[string]string),
		gProperties: make(map[string][]map[string]interface{}),
	}

	return aPipelineBuilderActivity2
}

func (a *PipelineBuilderActivity2) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *PipelineBuilderActivity2) Eval(context activity.Context) (done bool, err error) {

	log.Debug("[PipelineBuilderActivity2:Eval] entering ........ ")
	defer log.Debug("[PipelineBuilderActivity2:Eval] Exit ........ ")

	templateLibrary, gProperties, err := a.getTemplateLibrary(context)
	if err != nil {
		return false, err
	}

	applicationName, ok := context.GetInput(iApplicationName).(string)
	if !ok {
		return false, errors.New("Invalid Application Name ... ")
	}
	log.Debug("[PipelineBuilderActivity2:Eval]  Name : ", applicationName)

	serviceType, ok := context.GetInput(iServiceType).(string)
	if !ok {
		return false, errors.New("Invalid Service Type ... ")
	}
	log.Debug("[PipelineBuilderActivity2:Eval]  Name : ", serviceType)

	applicationPipelineDescriptor, ok := context.GetInput(iApplicationPipelineDescriptor).(map[string]interface{})
	if !ok {
		return false, errors.New("Invalid Application Pipeline Descriptor ... ")
	}
	log.Debug("[PipelineBuilderActivity2:Eval]  Pipeline Descriptor : ", applicationPipelineDescriptor)

	/*********************************
	        Construct Pipeline
	**********************************/

	var ports []interface{}
	descriptor := make(map[string]interface{})
	var appProperties []interface{}

	/* Create a new pipeline */
	pipeline := templateLibrary.GetPipeline()

	/* Declare notification listener */
	notificationListeners := map[string]interface{}{
		"ErrorHandler": make([]interface{}, 0),
	}
	log.Debug("[PipelineBuilderActivity2:Eval] Declare listener for ErrorHandler : ", notificationListeners)

	/* Add notifier for error handlers */
	notifier := templateLibrary.GetComponent(0, "Notifier", "Default", nil).(model.Notifier)
	pipeline.AddNotifier("ErrorHandler", notifier)

	/* Adding data source */
	log.Debug("[PipelineBuilderActivity2:Eval] Preparing datasource ......")
	sourceObj := applicationPipelineDescriptor["source"].(map[string]interface{})
	category, name := parseName(sourceObj["name"].(string))
	dataSource := templateLibrary.GetComponent(-1, category, name, extractProperties(sourceObj)).(model.DataSource)

	pipeline.SetDataSource(dataSource)
	/* If any server port defined */
	if nil != sourceObj[iPorts] {
		ports = sourceObj[iPorts].([]interface{})
	}

	/* Adding logics and find a runner*/
	log.Debug("[PipelineBuilderActivity2:Eval] Adding logics ......")
	var runner interface{}
	for key, value := range applicationPipelineDescriptor {
		switch key {
		case "logic":
			logicArray := value.([]interface{})
			normalFlow := make([]interface{}, 0)
			errorFlow := make([]interface{}, 0)

			isEventFlow := true
			for _, logic := range logicArray {
				logicObj := logic.(map[string]interface{})
				category, _ := parseName(logicObj["name"].(string))

				if "Error" == category {
					isEventFlow = false
				}

				if isEventFlow {
					normalFlow = append(normalFlow, logic)
				} else {
					errorFlow = append(errorFlow, logic)
				}
			}

			logicSN := 0
			for _, logic := range normalFlow {
				logicObj := logic.(map[string]interface{})
				category, name := parseName(logicObj["name"].(string))
				logic := templateLibrary.GetComponent(logicSN, category, name, extractProperties(logicObj)).(model.Logic)
				pipeline.AddNormalLogic(logic)

				if nil != logic.GetRunner() {
					runner = logic.GetRunner()
				}

				/* Add notifier for the cmponent which generate notification. */
				if nil != logic.GetNotificationBroker() {
					/* Add Notifier */
					brokerCategory, brokerName := parseName(logic.GetNotificationBroker().(string))
					notifier := templateLibrary.GetComponent(logicSN, brokerCategory, brokerName, nil).(model.Notifier)
					pipeline.AddNotifier(fmt.Sprintf("%s_%d", category, logicSN), notifier)
				}
				logicSN++
			}

			pipeline.AddNormalLogic(templateLibrary.GetComponent(logicSN, "Endcap", "Dummy", []interface{}{}).(model.Logic))
			logicSN++

			notificationListeners["ErrorHandler"] = append(notificationListeners["ErrorHandler"].([]interface{}), fmt.Sprintf("Error_%d", logicSN))
			if 0 != len(errorFlow) {
				for _, logic := range errorFlow {
					logicObj := logic.(map[string]interface{})
					category, name := parseName(logicObj["name"].(string))
					pipeline.AddErrorLogic(templateLibrary.GetComponent(logicSN, category, name, extractProperties(logicObj)).(model.Logic))
					logicSN++
				}
				pipeline.AddErrorLogic(templateLibrary.GetComponent(logicSN, "Endcap", "Dummy", []interface{}{}).(model.Logic))
			} else {
				pipeline.AddErrorLogic(templateLibrary.GetComponent(logicSN, "Error", "Default", []interface{}{}).(model.Logic))
			}

			log.Debug("[PipelineBuilderActivity2:Eval] Defalut listener for ErrorHandler : ", notificationListeners)

		case "extra":
			if nil == value {
				continue
			}
			extraArray := value.([]interface{})
			for _, property := range extraArray {
				name := util.GetPropertyElement("Name", property).(string)
				if !strings.HasPrefix(name, "App.") {
					gProperties = append(gProperties, map[string]interface{}{
						"Name":  name,
						"Value": util.GetPropertyElement("Value", property),
						"Type":  util.GetPropertyElement("Type", property),
					})
				} else if "App.NotificationListeners" == name {
					/* Get notification listeners from request */
					var listeners map[string]interface{}
					json.Unmarshal([]byte(util.GetPropertyElement("Value", property).(string)), &listeners)
					log.Debug("[PipelineBuilderActivity2:Eval] Notification listeners from request : ", listeners)
					/* Merge listeners */
					for key, value := range listeners {
						if nil == notificationListeners[key] {
							notificationListeners[key] = value
						} else {
							for _, name := range value.([]interface{}) {
								notificationListeners[key] = append(notificationListeners[key].([]interface{}), name)
							}
						}
					}
				}
			}
		}
	}
	log.Debug("[PipelineBuilderActivity2:Eval]  NotificationListeners : ", notificationListeners)
	pipeline.SetListeners(notificationListeners)

	descriptorString, _ := pipeline.Build()
	descriptor[oFlogoApplicationDescriptor] = string(descriptorString)

	/*********************************
	    Construct Dynamic Parameter
	**********************************/

	propertyContainer := pipeline.GetProperties()
	appProperties = applicationPipelineDescriptor["properties"].([]interface{})
	exist := make(map[string]bool)
	propertiesWithUniqueName, err := propertyContainer.GetReplacements()
	if nil != err {
		return false, err
	}
	for _, property := range propertiesWithUniqueName {
		log.Debug("[PipelineBuilderActivity2:Eval]  property : ", property)
		name := property.(map[string]interface{})["Name"].(string)
		/* duplication fillter */
		if !exist[name] {
			appProperties = append(appProperties, property)
			exist[name] = true
		}
	}
	for _, property := range appProperties {
		name := property.(map[string]interface{})["Name"].(string)
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
	log.Debug("[PipelineBuilderActivity2:Eval]  pathMapper : ", pathMapper)
	log.Debug("[PipelineBuilderActivity2:Eval]  defVariable : ", defVariable)
	log.Debug("[PipelineBuilderActivity2:Eval]  propertyPrefix : ", propertyPrefix)
	log.Debug("[PipelineBuilderActivity2:Eval]  appProperties : ", appProperties)
	log.Debug("[PipelineBuilderActivity2:Eval]  gProperties : ", gProperties)
	log.Debug("[PipelineBuilderActivity2:Eval]  ports : ", ports)

	switch serviceType {
	case "k8s":
		descriptor[oF1Properties], err = a.createK8sF1Properties(
			pathMapper,
			defVariable,
			propertyPrefix,
			appProperties,
			gProperties,
			ports,
		)
	default:
		descriptor[oF1Properties], err = a.createDockerF1Properties(
			pathMapper,
			defVariable,
			propertyPrefix,
			appProperties,
			gProperties,
			ports,
		)
	}

	if nil != err {
		return true, err
	}

	log.Debug("[PipelineBuilderActivity2:Eval]  Descriptor : ", descriptor)
	log.Debug("[PipelineBuilderActivity2:Eval]  PropertyNameDef : ", propertyContainer.GetPropertyNameDef())
	log.Debug("[PipelineBuilderActivity2:Eval]  Runner : ", runner)

	context.SetOutput(oDescriptor, descriptor)
	context.SetOutput(oPropertyNameDef, propertyContainer.GetPropertyNameDef())
	context.SetOutput(oRunner, runner)

	return true, nil
}

func parseName(fullname string) (string, string) {
	category := fullname[:strings.Index(fullname, ".")]
	name := fullname[strings.Index(fullname, ".")+1:]
	return category, name
}

func extractProperties(logicObj map[string]interface{}) []interface{} {
	log.Debug("[PipelineBuilderActivity2:extractProperties]  extractProperties : ", extractProperties)
	appProperties := make([]interface{}, 0)
	if nil != logicObj[iProperties] {
		for _, property := range logicObj[iProperties].([]interface{}) {
			log.Debug("[PipelineBuilderActivity2:extractProperties]  Name : ", util.GetPropertyElement("Name", property))
			appProperties = append(appProperties, map[string]interface{}{
				"Name":  util.GetPropertyElement("Name", property),
				"Value": util.GetPropertyElement("Value", property),
				"Type":  util.GetPropertyElement("Type", property),
			})
		}
	}
	return appProperties
}

func (a *PipelineBuilderActivity2) createDockerF1Properties(
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
	log.Debug("[PipelineBuilderActivity2:createDockerF1Properties]  description1 : ", description)

	for _, property := range gProperties {
		log.Debug("[PipelineBuilderActivity2:createDockerF1Properties]  property : ", property)
		/* nil will not be accepted */
		value, dtype, err := util.GetPropertyValue(property["Value"], property["Type"])
		if nil != err {
			return nil, err
		}

		if "String" == dtype {
			value = pathMapper.Replace(value.(string), defVariable)
			sValue := value.(string)
			if sValue[0] == '$' && sValue[len(sValue)-1] == '$' {
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
	log.Debug("[PipelineBuilderActivity2:createDockerF1Properties]  description2 : ", description)
	return description, nil
}

func (a *PipelineBuilderActivity2) createK8sF1Properties(
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

func (a *PipelineBuilderActivity2) getTemplateLibrary(ctx activity.Context) (*model.FlogoTemplateLibrary, []map[string]interface{}, error) {

	log.Debug("[PipelineBuilderActivity2:getTemplate] entering ........ ")
	defer log.Debug("[PipelineBuilderActivity2:getTemplate] exit ........ ")

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

func (a *PipelineBuilderActivity2) getVariableMapper(ctx activity.Context) (*kwr.KeywordMapper, map[string]string, error) {
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
