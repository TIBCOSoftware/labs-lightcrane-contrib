/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */

package airmodel

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"strconv"
	"strings"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

const (
	iPorts      = "ports"
	iProperties = "properties"
)

func BuildFlogoApp(
	template *FlogoTemplateLibrary,
	applicationName string,
	applicationPipelineDescriptor map[string]interface{},
	gProperties []interface{},
	config map[string]interface{},
) (descriptorString string, pipeline Pipeline, runner interface{}, ports []interface{}, replicas int, err error) {

	log.Info("[airmodel.BuildFlogoApp] entering ........ ")
	defer log.Info("[airmodel.BuildFlogoApp] Exit ........ ")

	if "" == applicationName {
		return "", Pipeline{}, nil, nil, -1, errors.New("Invalid Application Name ... ")
	}
	log.Info("[airmodel.BuildFlogoApp]  Name : ", applicationName)

	if nil == applicationPipelineDescriptor {
		return "", Pipeline{}, nil, nil, -1, errors.New("Invalid Application Pipeline Descriptor ... ")
	}
	log.Info("[airmodel.BuildFlogoApp]  Pipeline Descriptor : ", applicationPipelineDescriptor)

	log.Info("[airmodel.BuildFlogoApp] config = ", config)

	/*********************************
	        Construct Pipeline
	**********************************/

	/* Create a new pipeline */
	pipeline = template.GetPipeline()

	/* Declare notification listener */
	notificationListeners := map[string]interface{}{
		"ErrorHandler": make([]interface{}, 0),
	}
	log.Info("[airmodel.BuildFlogoApp] Declare listener for ErrorHandler : ", notificationListeners)

	/* Add notifier for error handlers */
	notifier := template.GetComponent(0, "Notifier", "Default", nil).(Notifier)
	pipeline.AddNotifier("ErrorHandler", notifier)

	/* Adding data source */
	log.Info("[airmodel.BuildFlogoApp] Preparing datasource ......")
	sourceObj := applicationPipelineDescriptor["source"].(map[string]interface{})
	category, name := parseName(sourceObj["name"].(string))
	dataSource := template.GetComponent(-1, category, name, extractProperties(sourceObj)).(DataSource)

	pipeline.SetDataSource(dataSource)
	/* If any server port defined */
	if nil != sourceObj[iPorts] {
		ports = sourceObj[iPorts].([]interface{})
	}

	/* Adding logics and find a runner*/
	log.Info("[airmodel.BuildFlogoApp] Adding logics ......")
	replicas = 1
	if nil != config["replicas"] {
		replicas = int(config["replicas"].(float64))
	}

	if nil != applicationPipelineDescriptor["logic"] {
		logicArray := applicationPipelineDescriptor["logic"].([]interface{})
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
			logic := template.GetComponent(logicSN, category, name, extractProperties(logicObj)).(Logic)
			pipeline.AddNormalLogic(logic)

			if nil != logic.GetRunner() {
				runner = logic.GetRunner()
			}

			/* Add notifier for the cmponent which generate notification. */
			if nil != logic.GetNotificationBroker() {
				/* Add Notifier */
				brokerCategory, brokerName := parseName(logic.GetNotificationBroker().(string))
				notifier := template.GetComponent(logicSN, brokerCategory, brokerName, nil).(Notifier)
				pipeline.AddNotifier(fmt.Sprintf("%s_%d", category, logicSN), notifier)
			}
			logicSN++
		}

		pipeline.AddNormalLogic(template.GetComponent(logicSN, "Endcap", "Dummy", []interface{}{}).(Logic))
		logicSN++

		notificationListeners["ErrorHandler"] = append(notificationListeners["ErrorHandler"].([]interface{}), fmt.Sprintf("Error_%d", logicSN))
		if 0 != len(errorFlow) {
			for _, logic := range errorFlow {
				logicObj := logic.(map[string]interface{})
				category, name := parseName(logicObj["name"].(string))
				pipeline.AddErrorLogic(template.GetComponent(logicSN, category, name, extractProperties(logicObj)).(Logic))
				logicSN++
			}
			pipeline.AddErrorLogic(template.GetComponent(logicSN, "Endcap", "Dummy", []interface{}{}).(Logic))
		} else {
			pipeline.AddErrorLogic(template.GetComponent(logicSN, "Error", "Default", []interface{}{}).(Logic))
		}
	}

	if nil != applicationPipelineDescriptor["extra"] {
		extraArray := applicationPipelineDescriptor["extra"].([]interface{})
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
				log.Info("[airmodel.BuildFlogoApp] Notification listeners from request : ", listeners)
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
			} // else if "App.Replicas" == name {
			//	replicas, _ = strconv.Atoi(util.GetPropertyElement("Value", property).(string))
			//}
		}
	}

	if nil != applicationPipelineDescriptor["properties"] {
		propertiesArray := applicationPipelineDescriptor["properties"].([]interface{})
		for index, property := range propertiesArray {
			log.Info("[airmodel.BuildFlogoApp] applicationPipelineDescriptor[\"properties\"] : index = ", index, ", property = ", property)
		}
		configByte, err := json.Marshal(config)
		if nil == err {
			applicationPipelineDescriptor["properties"] = append(propertiesArray, map[string]interface{}{
				"Name":  "App.Config",
				"Value": string(configByte),
				"Type":  "string",
			})

			if nil != config["HA"] {
				applicationPipelineDescriptor["properties"] = append(propertiesArray, map[string]interface{}{
					"Name":  "App.HA.Replicas",
					"Value": strconv.Itoa(int(config["HA"].(map[string]interface{})["replicas"].(float64))),
					"Type":  "string",
				})

				controllerPropertiesByte, err := json.Marshal(config["HA"].(map[string]interface{})["controllerProperties"])
				if nil == err {
					applicationPipelineDescriptor["properties"] = append(propertiesArray, map[string]interface{}{
						"Name":  "App.HA.Properties",
						"Value": string(controllerPropertiesByte),
						"Type":  "string",
					})
				}
			} else {
				log.Warnf("No HA setup in config.json")
			}
		}
	}

	log.Info("[airmodel.BuildFlogoApp]  NotificationListeners : ", notificationListeners)

	pipeline.SetListeners(notificationListeners)

	descriptorString, err = pipeline.Build()

	return descriptorString, pipeline, runner, ports, replicas, err
}

func parseName(fullname string) (string, string) {
	category := fullname[:strings.Index(fullname, ".")]
	name := fullname[strings.Index(fullname, ".")+1:]
	return category, name
}

func extractProperties(logicObj map[string]interface{}) []interface{} {
	log.Info("[airmodel.extractProperties]  logicObj : ", logicObj)
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

/* Contributes */

func NewContributes() Contributes {
	return Contributes{contributeMap: make(map[string]interface{})}
}

type Contributes struct {
	contributeMap map[string]interface{}
}

func (this *Contributes) Add(contributes interface{}) {
	switch contributes.(type) {
	case string:
		this.AddContributes(contributes.(string))
	case []interface{}:
		for _, contribute := range contributes.([]interface{}) {
			this.contributeMap[contribute.(map[string]interface{})["ref"].(string)] = contribute.(map[string]interface{})
		}
	}
}

func (this *Contributes) AddContributes(encodedContributeString string) {
	//log.Debug("encodedContributeString : " + encodedContributeString)
	contributeString, _ := b64.URLEncoding.DecodeString(encodedContributeString)
	var contributes interface{}
	if err := json.Unmarshal([]byte(contributeString), &contributes); err != nil {
		//panic(err)
		return
	}
	for _, contribute := range contributes.([]interface{}) {
		this.contributeMap[contribute.(map[string]interface{})["ref"].(string)] = contribute.(map[string]interface{})
	}
}

func (this *Contributes) GetString() string {
	contributeArray := make([]interface{}, 0)
	for _, contribute := range this.contributeMap {
		contributeArray = append(contributeArray, contribute)
	}
	contributeArrayBytes, _ := json.Marshal(contributeArray)
	contributeArrayString := b64.URLEncoding.EncodeToString(contributeArrayBytes)
	return contributeArrayString
}

func (this *Contributes) Clone() Contributes {
	return Contributes{
		contributeMap: util.DeepCopy(this.contributeMap).(map[string]interface{}),
	}
}

/* Properties */

func NewProperties() Properties {
	return Properties{
		appPropertiesByComponent: make([]interface{}, 0),
		componentSequence:        make([]interface{}, 0),
		properties:               make([]interface{}, 0),
		propertyMamingMap:        make([]interface{}, 0),
	}
}

type Properties struct {
	appPropertiesByComponent []interface{}
	componentSequence        []interface{}
	properties               []interface{}
	propertyMamingMap        []interface{}
}

func (this *Properties) Add(
	component string,
	properties []interface{},
	propertyMamingMap []interface{},
	newDefinedProperties []interface{},
	runtimeProperties []interface{}) {

	log.Info("(Properties.Add) component : ", component)
	log.Info("(Properties.Add) template properties : ", properties)
	log.Info("(Properties.Add) raw properties : ", propertyMamingMap)
	log.Info("(Properties.Add) new defined properties : ", newDefinedProperties)
	log.Info("(Properties.Add) runtime properties : ", runtimeProperties)

	this.appPropertiesByComponent = append(this.appPropertiesByComponent, runtimeProperties)

	mamingMap := make(map[string]interface{})
	componentName := fmt.Sprintf("%s_%s", component, "App.ComponentName")
	log.Info("(Properties.Add) componentName : ", componentName)
	foundName := false
	for index, property := range properties {
		this.properties = append(this.properties, property)
		if componentName == property.(map[string]interface{})["name"] {
			property.(map[string]interface{})["value"] = component
			log.Info("(Properties.Add) App.ComponentName defined : ", componentName)
			foundName = true
		}
		name := propertyMamingMap[index].(map[string]interface{})["name"].(string)
		leftBound := strings.Index(name, "${{")
		rightBound := strings.Index(name, "}}$") + 3
		if -1 < leftBound && leftBound < rightBound {
			name = fmt.Sprintf("%s%s", name[0:leftBound], name[rightBound:len(name)])
		}

		mamingMap[name] = property.(map[string]interface{})["name"]
		mamingMap[strings.ReplaceAll(name, ".", "_")] = property.(map[string]interface{})["name"]
	}
	if !foundName {
		this.properties = append(this.properties, map[string]interface{}{
			"name":  componentName,
			"value": component,
			"type":  "string",
		})
	}

	/* Directly set to component's default properties */
	if nil != newDefinedProperties {
		for _, property := range newDefinedProperties {
			this.properties = append(this.properties, property)
		}
	}

	this.componentSequence = append(this.componentSequence, component)
	this.propertyMamingMap = append(this.propertyMamingMap, mamingMap)
}

func (this *Properties) GetProperties() []interface{} {
	propertiesArray := make([]interface{}, 0)
	for _, property := range this.properties {
		log.Info("[Properties:GetProperties] Default property : ", property)
		propertiesArray = append(propertiesArray, property)
	}
	return propertiesArray
}

func (this *Properties) GetPropertyNameDef() map[string]interface{} {
	propertyNameDef := make(map[string]interface{})
	for index, component := range this.componentSequence {
		propertyNameDef[component.(string)] = this.propertyMamingMap[index]
	}
	return propertyNameDef
}

func (this *Properties) GetReplacements() ([]interface{}, error) {
	log.Info("(Properties:GetReplacements)  appPropertiesByComponent : ", this.appPropertiesByComponent)
	log.Info("(Properties:GetReplacements)  propertyMamingMap : ", this.propertyMamingMap)
	if len(this.propertyMamingMap) != len(this.appPropertiesByComponent) {
		log.Error("(Properties.GetReplacements) len(propertyMamingMap) : ", len(this.propertyMamingMap), ", len(appPropertiesByComponent) : ", len(this.appPropertiesByComponent))
		return nil, errors.New("Component size doesn't match which in the propertyMamingMap!!!")
	}
	appProperties := make([]interface{}, 0)
	/* loop for component in processing order */
	for index, componentProperties := range this.appPropertiesByComponent {
		log.Info("(Properties.GetReplacements) index : ", index)
		for _, property := range componentProperties.([]interface{}) {
			name := property.(map[string]interface{})["Name"].(string)
			log.Info("app property name: ", name)
			if index < len(this.propertyMamingMap) && nil != this.propertyMamingMap[index].(map[string]interface{})[name] {
				name = this.propertyMamingMap[index].(map[string]interface{})[name].(string)
				log.Info("app property name after: ", name)
				property.(map[string]interface{})["Name"] = this.propertyMamingMap[index].(map[string]interface{})[property.(map[string]interface{})["Name"].(string)]
				appProperties = append(appProperties, property)
			}
		}
	}
	return appProperties, nil
}

func (this *Properties) Clone() Properties {
	return Properties{
		componentSequence: util.DeepCopy(this.componentSequence).([]interface{}),
		properties:        util.DeepCopy(this.properties).([]interface{}),
		propertyMamingMap: util.DeepCopy(this.propertyMamingMap).([]interface{}),
	}
}

/* Imports */
func NewImports() Imports {
	return Imports{imports: make(map[string]interface{})}
}

type Imports struct {
	imports map[string]interface{}
}

func (this *Imports) Add(imports []interface{}) {
	log.Debug(imports)
	for _, anImport := range imports {
		this.imports[anImport.(string)] = anImport
	}
	log.Debug(this)
}

func (this *Imports) GetImports() []interface{} {
	importsArray := make([]interface{}, 0)
	for _, anImport := range this.imports {
		importsArray = append(importsArray, anImport)
	}
	return importsArray
}

func (this *Imports) Clone() Imports {
	return Imports{
		imports: util.DeepCopy(this.imports).(map[string]interface{}),
	}
}

/* Connections */
func NewConnections() Connections {
	return Connections{connections: make(map[string]interface{})}
}

type Connections struct {
	connections map[string]interface{}
}

func (this *Connections) Add(connections interface{}) {
	log.Debug(connections)
	if nil != connections {
		for id, connection := range connections.(map[string]interface{}) {
			this.connections[id] = connection
		}
	}
	log.Debug(connections)
}

func (this *Connections) GetConnections() interface{} {
	return this.connections
}

func (this *Connections) Clone() Connections {
	return Connections{
		connections: util.DeepCopy(this.connections).(map[string]interface{}),
	}
}
