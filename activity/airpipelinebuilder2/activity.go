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

	kwr "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/keywordreplace"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/log"

	model "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/airmodel"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

const (
	iApplicationProperties      = "ApplicationProperties"
	iPorts                      = "ports"
	iProperties                 = "properties"
	iPropertyPrefix             = "PropertyPrefix"
	iVariable                   = "Variables"
	oFlogoApplicationDescriptor = "FlogoDescriptor"
	oF1Properties               = "F1Properties"
	oDescriptor                 = "Descriptor"
	oPropertyNameDef            = "PropertyNameDef"
	oRunner                     = "Runner"
	oVariable                   = "Variables"
)

type Settings struct {
	TemplateFolder string `md:"TemplateFolder"`
	LeftToken      string `md:"leftToken"`
	RightToken     string `md:"rightToken"`
	VariablesDef   string `md:"variablesDef"`
	Properties     string `md:"Properties"`
}

type Input struct {
	ApplicationName               string                 `md:"ApplicationName"`
	ApplicationPipelineDescriptor map[string]interface{} `md:"AirDescriptor"`
	ServiceType                   string                 `md:"ServiceType"`
	PropertyPrefix                string                 `md:"PropertyPrefix"`
	Variable                      map[string]interface{} `md:"Variables"`
}

type Output struct {
	Descriptor      map[string]interface{} `md:"Descriptor"`
	PropertyNameDef map[string]interface{} `md:"PropertyNameDef"`
	Runner          string                 `md:"Runner"`
	Variable        map[string]interface{} `md:"Variables"`
}

func (i *Input) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"ApplicationName": i.ApplicationName,
		"AirDescriptor":   i.ApplicationPipelineDescriptor,
		"ServiceType":     i.ServiceType,
		"PropertyPrefix":  i.PropertyPrefix,
		"Variable":        i.Variable,
	}
}

func (i *Input) FromMap(values map[string]interface{}) error {
	ok := true
	i.ApplicationName, ok = values["ApplicationName"].(string)
	if !ok {
		return errors.New("Illegal ApplicationName type, expect string.")
	}
	i.ApplicationPipelineDescriptor, ok = values["AirDescriptor"].(map[string]interface{})
	if !ok {
		return errors.New("Illegal ApplicationPipelineDescriptor type, expect map[string]interface{}.")
	}
	i.ServiceType, ok = values["ServiceType"].(string)
	if !ok {
		return errors.New("Illegal ServiceType type, expect string.")
	}
	i.PropertyPrefix, ok = values["PropertyPrefix"].(string)
	if !ok {
		return errors.New("Illegal PropertyPrefix type, expect string.")
	}
	i.Variable, ok = values["Variable"].(map[string]interface{})
	if !ok {
		return errors.New("Illegal Variable type, expect map[string]interface{}.")
	}
	return nil
}

func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Descriptor":      o.Descriptor,
		"PropertyNameDef": o.PropertyNameDef,
		"Runner":          o.Runner,
		"Variable":        o.Variable,
	}
}

func (o *Output) FromMap(values map[string]interface{}) error {
	ok := true
	o.Descriptor, ok = values["Descriptor"].(map[string]interface{})
	if !ok {
		return errors.New("Illegal Descriptor type, expect map[string]interface{}.")
	}
	o.PropertyNameDef, ok = values["PropertyNameDef"].(map[string]interface{})
	if !ok {
		return errors.New("Illegal PropertyNameDef type, expect map[string]interface{}.")
	}
	o.Runner, ok = values["Runner"].(string)
	if !ok {
		return errors.New("Illegal Runner type, expect string.")
	}
	o.Variable, ok = values["Variable"].(map[string]interface{})
	if !ok {
		return errors.New("Illegal Variable type, expect map[string]interface{}.")
	}
	return nil
}

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

func init() {
	_ = activity.Register(&Activity{}, New)
}

type Activity struct {
	template    *model.FlogoTemplateLibrary
	pathMapper  *kwr.KeywordMapper
	variables   map[string]string
	gProperties []map[string]interface{}
}

func New(ctx activity.InitContext) (activity.Activity, error) {
	settings := &Settings{}
	err := metadata.MapToStruct(ctx.Settings(), settings, true)
	if err != nil {
		return nil, err
	}

	// Build templates
	templateFolder := settings.TemplateFolder
	if "" == templateFolder {
		return nil, activity.NewError("Template is not configured", "PipelineBuilder-4002", nil)
	}
	templateLib, err := model.NewFlogoTemplateLibrary(templateFolder)
	if nil != err {
		return nil, err
	}

	// Build group properties
	gProperties := make([]map[string]interface{}, 0)
	if "" != settings.Properties {
		var gPropertiesSetting []interface{}
		err := json.Unmarshal([]byte(settings.Properties), &gPropertiesSetting)
		if nil == err {
			for _, gProperty := range gPropertiesSetting {
				gProperties = append(gProperties, gProperty.(map[string]interface{}))
			}
		}
	}

	// Build variables
	variables := make(map[string]string)
	if "" != settings.VariablesDef {
		var variablesDef []interface{}
		err := json.Unmarshal([]byte(settings.VariablesDef), &variablesDef)
		if nil == err && nil != variablesDef {
			for _, variableDef := range variablesDef {
				variableInfo := variableDef.(map[string]interface{})
				variables[variableInfo["Name"].(string)] = variableInfo["Type"].(string)
			}
		}
	}
	// Build pathMapper
	lefttoken := settings.LeftToken
	if "" == lefttoken {
		return nil, errors.New("LeftToken not defined!")
	}
	righttoken := settings.RightToken
	if "" == righttoken {
		return nil, errors.New("RightToken not defined!")
	}
	mapper := kwr.NewKeywordMapper("", lefttoken, righttoken)

	activity := &Activity{
		template:    templateLib,
		pathMapper:  mapper,
		variables:   variables,
		gProperties: gProperties,
	}

	return activity, nil
}

func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

func (a *Activity) Eval(ctx activity.Context) (done bool, err error) {

	log := ctx.Logger()
	log.Info("[PipelineBuilderActivity2:Eval] entering ........ ")
	defer log.Info("[PipelineBuilderActivity2:Eval] Exit ........ ")

	input := &Input{}
	ctx.GetInputObject(input)

	gProperties := util.DeepCopy(a.gProperties).([]map[string]interface{})

	applicationName := input.ApplicationName
	if "" == applicationName {
		return false, errors.New("Invalid Application Name ... ")
	}
	log.Info("[PipelineBuilderActivity2:Eval]  Name : ", applicationName)

	serviceType := input.ServiceType
	if "" == serviceType {
		return false, errors.New("Invalid Service Type ... ")
	}
	log.Info("[PipelineBuilderActivity2:Eval]  Name : ", serviceType)

	applicationPipelineDescriptor := input.ApplicationPipelineDescriptor
	if nil == applicationPipelineDescriptor {
		return false, errors.New("Invalid Application Pipeline Descriptor ... ")
	}
	log.Info("[PipelineBuilderActivity2:Eval]  Pipeline Descriptor : ", applicationPipelineDescriptor)

	/*********************************
	        Construct Pipeline
	**********************************/

	var ports []interface{}
	descriptor := make(map[string]interface{})
	var appProperties []interface{}

	/* Create a new pipeline */
	pipeline := a.template.GetPipeline()

	/* Declare notification listener */
	notificationListeners := map[string]interface{}{
		"ErrorHandler": make([]interface{}, 0),
	}
	log.Info("[PipelineBuilderActivity2:Eval] Declare listener for ErrorHandler : ", notificationListeners)

	/* Add notifier for error handlers */
	notifier := a.template.GetComponent(0, "Notifier", "Default", nil).(model.Notifier)
	pipeline.AddNotifier("ErrorHandler", notifier)

	/* Adding data source */
	log.Info("[PipelineBuilderActivity2:Eval] Preparing datasource ......")
	sourceObj := applicationPipelineDescriptor["source"].(map[string]interface{})
	category, name := parseName(sourceObj["name"].(string))
	dataSource := a.template.GetComponent(-1, category, name, extractProperties(log, sourceObj)).(model.DataSource)

	pipeline.SetDataSource(dataSource)
	/* If any server port defined */
	if nil != sourceObj[iPorts] {
		ports = sourceObj[iPorts].([]interface{})
	}

	/* Adding logics and find a runner*/
	log.Info("[PipelineBuilderActivity2:Eval] Adding logics ......")
	var runner interface{}
	replicas := 1
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
				logic := a.template.GetComponent(logicSN, category, name, extractProperties(log, logicObj)).(model.Logic)
				pipeline.AddNormalLogic(logic)

				if nil != logic.GetRunner() {
					runner = logic.GetRunner()
				}

				/* Add notifier for the cmponent which generate notification. */
				if nil != logic.GetNotificationBroker() {
					/* Add Notifier */
					brokerCategory, brokerName := parseName(logic.GetNotificationBroker().(string))
					notifier := a.template.GetComponent(logicSN, brokerCategory, brokerName, nil).(model.Notifier)
					pipeline.AddNotifier(fmt.Sprintf("%s_%d", category, logicSN), notifier)
				}
				logicSN++
			}

			pipeline.AddNormalLogic(a.template.GetComponent(logicSN, "Endcap", "Dummy", []interface{}{}).(model.Logic))
			logicSN++

			notificationListeners["ErrorHandler"] = append(notificationListeners["ErrorHandler"].([]interface{}), fmt.Sprintf("Error_%d", logicSN))
			if 0 != len(errorFlow) {
				for _, logic := range errorFlow {
					logicObj := logic.(map[string]interface{})
					category, name := parseName(logicObj["name"].(string))
					pipeline.AddErrorLogic(a.template.GetComponent(logicSN, category, name, extractProperties(log, logicObj)).(model.Logic))
					logicSN++
				}
				pipeline.AddErrorLogic(a.template.GetComponent(logicSN, "Endcap", "Dummy", []interface{}{}).(model.Logic))
			} else {
				pipeline.AddErrorLogic(a.template.GetComponent(logicSN, "Error", "Default", []interface{}{}).(model.Logic))
			}

			log.Info("[PipelineBuilderActivity2:Eval] Defalut listener for ErrorHandler : ", notificationListeners)

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
					log.Info("[PipelineBuilderActivity2:Eval] Notification listeners from request : ", listeners)
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
				} else if "App.Replicas" == name {
					replicas = int(util.GetPropertyElement("Value", property).(float64))
				}
			}
		}
	}
	log.Info("[PipelineBuilderActivity2:Eval]  NotificationListeners : ", notificationListeners)
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
		log.Info("[PipelineBuilderActivity2:Eval] Dynamic property : ", property)
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

	defVariable := input.Variable
	propertyPrefix := a.pathMapper.Replace(input.PropertyPrefix, defVariable)

	log.Info("[PipelineBuilderActivity2:Eval]  pathMapper : ", a.pathMapper)
	log.Info("[PipelineBuilderActivity2:Eval]  defVariable : ", defVariable)
	log.Info("[PipelineBuilderActivity2:Eval]  propertyPrefix : ", propertyPrefix)
	log.Info("[PipelineBuilderActivity2:Eval]  appProperties : ", appProperties)
	log.Info("[PipelineBuilderActivity2:Eval]  gProperties : ", gProperties)
	log.Info("[PipelineBuilderActivity2:Eval]  ports : ", ports)
	log.Info("[PipelineBuilderActivity2:Eval]  replicas : ", replicas)

	switch serviceType {
	case "k8s":
		descriptor[oF1Properties], err = a.createK8sF1Properties(
			log,
			a.pathMapper,
			defVariable,
			propertyPrefix,
			appProperties,
			gProperties,
			ports,
			replicas,
		)
	default:
		descriptor[oF1Properties], err = a.createDockerF1Properties(
			log,
			a.pathMapper,
			defVariable,
			propertyPrefix,
			appProperties,
			gProperties,
			ports,
			replicas,
		)
	}

	if nil != err {
		return true, err
	}

	log.Info("[PipelineBuilderActivity2:Eval]  Descriptor : ", descriptor)
	log.Info("[PipelineBuilderActivity2:Eval]  PropertyNameDef : ", propertyContainer.GetPropertyNameDef())
	log.Info("[PipelineBuilderActivity2:Eval]  Runner : ", runner)
	log.Info("[PipelineBuilderActivity2:Eval]  Variables : ", defVariable)

	ctx.SetOutput(oDescriptor, descriptor)
	ctx.SetOutput(oPropertyNameDef, propertyContainer.GetPropertyNameDef())
	ctx.SetOutput(oRunner, runner)
	ctx.SetOutput(oVariable, defVariable)

	return true, nil
}

func parseName(fullname string) (string, string) {
	category := fullname[:strings.Index(fullname, ".")]
	name := fullname[strings.Index(fullname, ".")+1:]
	return category, name
}

func extractProperties(log log.Logger, logicObj map[string]interface{}) []interface{} {
	log.Info("[PipelineBuilderActivity2:extractProperties]  logicObj : ", logicObj)
	appProperties := make([]interface{}, 0)
	if nil != logicObj[iProperties] {
		for _, property := range logicObj[iProperties].([]interface{}) {
			log.Info("[PipelineBuilderActivity2:extractProperties]  Name : ", util.GetPropertyElement("Name", property))
			appProperties = append(appProperties, map[string]interface{}{
				"Name":  util.GetPropertyElement("Name", property),
				"Value": util.GetPropertyElement("Value", property),
				"Type":  util.GetPropertyElement("Type", property),
			})
		}
	}
	return appProperties
}

func (a *Activity) createDockerF1Properties(
	log log.Logger,
	pathMapper *kwr.KeywordMapper,
	defVariable map[string]interface{},
	propertyPrefix string,
	appProperties []interface{},
	gProperties []map[string]interface{},
	ports []interface{},
	replica int,
) (interface{}, error) {

	description := make([]interface{}, 0)
	mainDescription := map[string]interface{}{
		"Group": "main",
		"Value": make([]interface{}, 0),
	}
	description = append(description, mainDescription)
	log.Info("[PipelineBuilderActivity2:createDockerF1Properties]  description1 : ", description)

	for _, property := range gProperties {
		log.Info("[PipelineBuilderActivity2:createDockerF1Properties]  property : ", property)
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
	log.Info("[PipelineBuilderActivity2:createDockerF1Properties] docker-compose description : ", description)
	return description, nil
}

func (a *Activity) createK8sF1Properties(
	log log.Logger,
	pathMapper *kwr.KeywordMapper,
	defVariable map[string]interface{},
	propertyPrefix string,
	appProperties []interface{},
	gProperties []map[string]interface{},
	ports []interface{},
	replicas int,
) (interface{}, error) {
	groupProperties := make(map[string]interface{})
	for _, property := range gProperties {
		name := util.GetPropertyElementAsString("Name", property)
		log.Info("[PipelineBuilderActivity2:createK8sF1Properties] name : ", name)
		if 0 < strings.Index(name, "_") {
			group := name[0:strings.Index(name, "_")]
			log.Info("[PipelineBuilderActivity2:createK8sF1Properties] has group name : ", group)
			if nil == groupProperties[group] {
				groupProperties[group] = make([]interface{}, 0)
			}
			name = name[strings.Index(name, "_")+1 : len(name)]
			property["Name"] = name
			groupProperties[group] = append(groupProperties[group].([]interface{}), property)
		} else {
			log.Info("[PipelineBuilderActivity2:createK8sF1Properties] has no group name! ")
		}
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

	// Add replicas
	if 1 < replicas {
		mainDescription["Value"] = append(mainDescription["Value"].([]interface{}), map[string]interface{}{
			"Name":  "spec.replicas",
			"Value": replicas,
			"Type":  "string",
		})
	}

	// Add preconfigured properties
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

	// Add pipeline parameters
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

	log.Info("[PipelineBuilderActivity2:createK8sF1Properties] k8s description : ", description)
	return description, nil
}
