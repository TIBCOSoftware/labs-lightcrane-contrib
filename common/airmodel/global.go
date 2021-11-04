/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */

package model

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

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
		componentSequence: make([]interface{}, 0),
		properties:        make([]interface{}, 0),
		propertyMamingMap: make([]interface{}, 0),
	}
}

type Properties struct {
	componentSequence []interface{}
	properties        []interface{}
	propertyMamingMap []interface{}
}

func (this *Properties) Add(component string, properties []interface{}, propertyMamingMap []interface{}, newDefinedProperties []interface{}) {
	mamingMap := make(map[string]interface{})
	componentName := fmt.Sprintf("%s_%s", component, "App.ComponentName")
	foundName := false
	for index, property := range properties {
		this.properties = append(this.properties, property)
		if componentName == property.(map[string]interface{})["name"] {
			property.(map[string]interface{})["value"] = component
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

func (this *Properties) GetReplacements(appPropertiesByComponent []interface{}) []interface{} {
	log.Debug("(Properties.GetReplacements) appPropertiesByComponent : ", appPropertiesByComponent)
	appProperties := make([]interface{}, 0)
	/* loop for component in processing order */
	for index, componentProperties := range appPropertiesByComponent {
		log.Debug("(Properties.GetReplacements) index : ", index)
		for _, property := range componentProperties.([]interface{}) {
			name := property.(map[string]interface{})["Name"].(string)
			log.Debug("app property name: ", name)
			if nil != this.propertyMamingMap[index].(map[string]interface{})[name] {
				name = this.propertyMamingMap[index].(map[string]interface{})[name].(string)
				log.Debug("app property name after: ", name)
				property.(map[string]interface{})["Name"] = this.propertyMamingMap[index].(map[string]interface{})[property.(map[string]interface{})["Name"].(string)]
				appProperties = append(appProperties, property)
			}
		}
	}
	return appProperties
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
