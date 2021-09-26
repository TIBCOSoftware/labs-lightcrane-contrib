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

package airpipelinebuilder

import (
	"encoding/json"
	"errors"

	"fmt"
	"strings"
	"sync"

	kwr "github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/keywordreplace"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"

	//	"github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/objectbuilder"
	model "github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/airmodel"
	"github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/util"
)

var log = logger.GetLogger("tibco-model-ops-pipelinebuilder")

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
)

type PipelineBuilderActivity struct {
	metadata    *activity.Metadata
	mux         sync.Mutex
	templates   map[string]*model.FlogoTemplateLibrary
	pathMappers map[string]*kwr.KeywordMapper
	variables   map[string]map[string]string
	gProperties map[string][]map[string]interface{}
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aPipelineBuilderActivity := &PipelineBuilderActivity{
		metadata:    metadata,
		templates:   make(map[string]*model.FlogoTemplateLibrary),
		pathMappers: make(map[string]*kwr.KeywordMapper),
		variables:   make(map[string]map[string]string),
		gProperties: make(map[string][]map[string]interface{}),
	}

	return aPipelineBuilderActivity
}

func (a *PipelineBuilderActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *PipelineBuilderActivity) Eval(context activity.Context) (done bool, err error) {

	log.Info("[PipelineBuilderActivity:Eval] entering ........ ")

	templateLibrary, gProperties, err := a.getTemplateLibrary(context)
	if err != nil {
		return false, err
	}

	applicationName, ok := context.GetInput(iApplicationName).(string)
	if !ok {
		return false, errors.New("Invalid Application Name ... ")
	}
	log.Info("[PipelineBuilderActivity:Eval]  Name : ", applicationName)

	serviceType, ok := context.GetInput(iServiceType).(string)
	if !ok {
		return false, errors.New("Invalid Service Type ... ")
	}
	log.Info("[PipelineBuilderActivity:Eval]  Name : ", serviceType)

	applicationPipelineDescriptorStr, ok := context.GetInput(iApplicationPipelineDescriptor).(string)
	if !ok {
		return false, errors.New("Invalid Application Pipeline Descriptor ... ")
	}
	log.Info("[PipelineBuilderActivity:Eval]  Pipeline Descriptor : ", applicationPipelineDescriptorStr)

	var applicationPipelineDescriptor map[string]interface{}
	json.Unmarshal([]byte(applicationPipelineDescriptorStr), &applicationPipelineDescriptor)
	if nil != err {
		return true, err
	}

	/*********************************
	        Construct Pipeline
	**********************************/

	var ports []interface{}
	descriptor := make(map[string]interface{})
	appPropertiesByComponent := make([]interface{}, 0)
	var appProperties []interface{}

	/* Create a new pipeline */

	pipeline := templateLibrary.GetPipeline()

	/* Adding data source */

	sourceObj := applicationPipelineDescriptor["source"].(map[string]interface{})
	longname := sourceObj["name"].(string)
	category := longname[:strings.Index(longname, ".")]
	name := longname[strings.Index(longname, ".")+1:]
	dataSource := templateLibrary.GetComponent(-1, category, name).(model.DataSource)
	pipeline.SetDataSource(dataSource)
	/* If any server port defined */
	if nil != sourceObj[iPorts] {
		ports = sourceObj[iPorts].([]interface{})
	}
	/* Extrace Daynamic Parameter From DataSource */
	appProperties = make([]interface{}, 0)
	if nil != sourceObj[iProperties] {
		for _, property := range sourceObj[iProperties].([]interface{}) {
			appProperties = append(appProperties, map[string]interface{}{
				"Name":  util.GetPropertyElement("Name", property),
				"Value": util.GetPropertyElement("Value", property),
				"Type":  util.GetPropertyElement("Type", property),
			})
		}
	}
	appPropertiesByComponent = append(appPropertiesByComponent, appProperties)

	/* Adding logics */

	for key, value := range applicationPipelineDescriptor {
		switch key {
		case "logic":
			logicArray := value.([]interface{})
			for index, logic := range logicArray {
				logicObj := logic.(map[string]interface{})
				longname := logicObj["name"].(string)
				category := longname[:strings.Index(longname, ".")]
				name := longname[strings.Index(longname, ".")+1:]
				logic := templateLibrary.GetComponent(index, category, name).(model.Logic)
				pipeline.AddLogic(logic)
				if "Rule.Default" == longname || "Rule.Expression" == longname {
					/* Add Notifier */
					notifier := templateLibrary.GetComponent(index, "Notifier", "Default").(model.Notifier)
					pipeline.AddNotifier(fmt.Sprintf("%s_%d", category, index), notifier)
				}

				/* Extrace Daynamic Parameter From Logic */
				appProperties = make([]interface{}, 0)
				if nil != logicObj[iProperties] {
					for _, property := range logicObj[iProperties].([]interface{}) {
						appProperties = append(appProperties, map[string]interface{}{
							"Name":  util.GetPropertyElement("Name", property),
							"Value": util.GetPropertyElement("Value", property),
							"Type":  util.GetPropertyElement("Type", property),
						})
					}
				}
				appPropertiesByComponent = append(appPropertiesByComponent, appProperties)
			}
			/* Dummy
						 {
			                "name": "Filter.Dummy",
			                "properties": [
			                    {
			                        "Name": "Logging.LogLevel",
			                        "Value": "DEBUG"
			                    }
			                ]
			            }
			*/
			longname := "Filter.Dummy"
			category := longname[:strings.Index(longname, ".")]
			name := longname[strings.Index(longname, ".")+1:]
			logic := templateLibrary.GetComponent(len(logicArray), category, name).(model.Logic)
			pipeline.AddLogic(logic)

			appPropertiesByComponent = append(appPropertiesByComponent,
				[]interface{}{
					map[string]interface{}{
						"Name":  "Logging.LogLevel",
						"Value": "DEBUG",
					},
				},
			)

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
					var listeners map[string]interface{}
					json.Unmarshal([]byte(util.GetPropertyElement("Value", property).(string)), &listeners)
					pipeline.SetListeners(listeners)
				}
			}
		}
	}

	descriptorString, _ := pipeline.Build()
	descriptor[oFlogoApplicationDescriptor] = string(descriptorString)

	/*********************************
	    Construct Dynamic Parameter
	**********************************/

	propertyContainer := pipeline.GetProperties()
	appProperties = applicationPipelineDescriptor["properties"].([]interface{})
	exist := make(map[string]bool)
	for _, property := range propertyContainer.GetReplacements(appPropertiesByComponent) {
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
	log.Info("[PipelineBuilderActivity:Eval]  pathMapper : ", pathMapper)
	log.Info("[PipelineBuilderActivity:Eval]  defVariable : ", defVariable)
	log.Info("[PipelineBuilderActivity:Eval]  propertyPrefix : ", propertyPrefix)
	log.Info("[PipelineBuilderActivity:Eval]  appProperties : ", appProperties)
	log.Info("[PipelineBuilderActivity:Eval]  gProperties : ", gProperties)
	log.Info("[PipelineBuilderActivity:Eval]  ports : ", ports)

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

	log.Info("[PipelineBuilderActivity:Eval]  Descriptor : ", descriptor)
	log.Info("[PipelineBuilderActivity:Eval]  PropertyNameDef : ", propertyContainer.GetPropertyNameDef())

	context.SetOutput(oDescriptor, descriptor)
	context.SetOutput(oPropertyNameDef, propertyContainer.GetPropertyNameDef())

	log.Info("[PipelineBuilderActivity:Eval] Exit ........ ")

	return true, nil
}

func (a *PipelineBuilderActivity) createDockerF1Properties(
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
	log.Info("[PipelineBuilderActivity:createDockerF1Properties]  description1 : ", description)

	for _, property := range gProperties {
		log.Info("[PipelineBuilderActivity:createDockerF1Properties]  property : ", property)
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
	log.Info("[PipelineBuilderActivity:createDockerF1Properties]  description2 : ", description)
	return description, nil
}

func (a *PipelineBuilderActivity) createK8sF1Properties(
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

func (a *PipelineBuilderActivity) getTemplateLibrary(ctx activity.Context) (*model.FlogoTemplateLibrary, []map[string]interface{}, error) {

	log.Info("[PipelineBuilderActivity:getTemplate] entering ........ ")
	defer log.Info("[PipelineBuilderActivity:getTemplate] exit ........ ")

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

func (a *PipelineBuilderActivity) getVariableMapper(ctx activity.Context) (*kwr.KeywordMapper, map[string]string, error) {
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
			log.Info("Processing handlers : variablesDef = ", variablesDef)
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
