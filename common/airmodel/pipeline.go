/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */

package model

import (
	"encoding/json"
	"fmt"

	"github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/objectbuilder"
	"github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/util"
	"github.com/TIBCOSoftware/flogo-lib/logger"
)

var log = logger.GetLogger("tibco-f1-pipeline-model")

/* Pipeline Class */

func NewPipeline(applicationName string, filename string) (Pipeline, error) {
	data, err := FromFile(filename)
	imports := NewImports()
	imports.Add(data["imports"].([]interface{}))
	contributes := NewContributes()
	contributes.Add(data["contrib"])

	return Pipeline{
		applicationName: applicationName,
		data:            data,
		contributes:     contributes,
		properties:      NewProperties(),
		connections:     NewConnections(),
		imports:         imports,
	}, err
}

type Pipeline struct {
	applicationName string
	data            map[string]interface{} // Application template
	dataSource      DataSource             // Main flow trigger
	logics          []Logic                // All subflow
	notifiers       map[string]Notifier
	listeners       map[string]interface{}
	contributes     Contributes
	properties      Properties
	connections     Connections
	imports         Imports
}

func (this *Pipeline) Build() (string, error) {

	flogoFlows := make([]interface{}, 0)
	this.dataSource.Build(fmt.Sprintf("%s_%d", this.logics[0].GetCategory(), 0))
	this.contributes.Add(this.dataSource.GetContribution())
	this.imports.Add(this.dataSource.GetImports())
	this.properties.Add(this.dataSource.GetID(), this.dataSource.GetProperties(), this.dataSource.GetRawProperties(), nil)
	this.connections.Add(this.dataSource.GetConnections())
	flogoFlows = append(flogoFlows, this.dataSource.GetResource())

	for index, logic := range this.logics {
		if index < len(this.logics)-1 {
			logic.Build(fmt.Sprintf("%s_%d", this.logics[index+1].GetCategory(), index+1), false)
		} else {
			logic.Build(fmt.Sprintf("%s_%d", "", -1), true)
		}
		this.contributes.Add(logic.GetContribution())
		this.imports.Add(logic.GetImports())
		isListener := false
		for _, listenerGroup := range this.listeners {
			fmt.Println("(Pipeline.Build)  listenerGroup = ", listenerGroup)
			for _, listener := range listenerGroup.([]interface{}) {
				fmt.Println("(Pipeline.Build)    listener = ", listener)
				if listener == logic.GetID() {
					isListener = true
				}
			}
		}
		properties := []interface{}{
			map[string]interface{}{
				"name":  fmt.Sprintf("%s_App.IsListener", logic.GetID()),
				"type":  "boolean",
				"value": isListener,
			},
		}
		this.properties.Add(logic.GetID(), logic.GetProperties(), logic.GetRawProperties(), properties)
		this.connections.Add(logic.GetConnections())
		flogoFlows = append(flogoFlows, logic.GetResource())
	}

	elementMap := make(map[string]*objectbuilder.Element)

	elementMap["root.name"] = objectbuilder.NewElement(
		"name",
		this.applicationName,
		"string",
	)

	triggers := this.dataSource.GetTriggers()
	for ID, notifier := range this.notifiers {
		for _, trigger := range notifier.GetTriggers(ID, this.listeners) {
			triggers = append(triggers, trigger)
			this.imports.Add(notifier.GetImports())
		}
	}

	elementMap["root.triggers[]"] = objectbuilder.NewElement(
		"triggers",
		triggers,
		"[]interface {}",
	)

	elementMap["root.resources[]"] = objectbuilder.NewElement(
		"resources",
		flogoFlows,
		"[]interface {}",
	)

	/* populate imports */
	elementMap["root.imports[]"] = objectbuilder.NewElement(
		"imports",
		this.imports.GetImports(),
		"[]interface {}",
	)

	/* populate contributes */
	elementMap["root.contrib"] = objectbuilder.NewElement(
		"contrib",
		this.contributes.GetString(),
		"string",
	)

	/* populate properties */
	elementMap["root.properties[]"] = objectbuilder.NewElement(
		"properties",
		this.properties.GetProperties(),
		"[]interface {}",
	)

	/* populate connections */
	elementMap["root.connections"] = objectbuilder.NewElement(
		"connections",
		this.connections.GetConnections(),
		"map[string]interface {}",
	)

	builder := objectbuilder.NewFlogoAppBuilder(elementMap)
	newPipeline := builder.Build(builder, this.data)
	jsondata, err := json.Marshal(newPipeline)
	if nil != err {
		//fmt.Println("Unable to serialize object, reason : ", err.Error())
		return "", err
	}
	return string(jsondata), err
}

func (this Pipeline) Clone() Pipeline {
	return Pipeline{
		applicationName: this.applicationName,
		data:            util.DeepCopy(this.data).(map[string]interface{}),
		contributes:     this.contributes.Clone(),
		properties:      this.properties.Clone(),
		imports:         this.imports.Clone(),
		connections:     this.connections.Clone(),
	}
}

func (this *Pipeline) SetDataSource(source DataSource) {
	this.dataSource = source
}

func (this *Pipeline) AddLogic(logic Logic) {
	this.logics = append(this.logics, logic)
}

func (this *Pipeline) AddNotifier(ID string, notifier Notifier) {
	if nil == this.notifiers {
		this.notifiers = make(map[string]Notifier)
	}
	this.notifiers[ID] = notifier
}

func (this *Pipeline) SetListeners(listeners map[string]interface{}) {
	this.listeners = listeners
}

func (this *Pipeline) GetDataSource() DataSource {
	return this.dataSource
}

func (this *Pipeline) GetData() map[string]interface{} {
	return this.data
}

func (this *Pipeline) GetContributes() Contributes {
	return this.contributes
}

func (this *Pipeline) GetProperties() Properties {
	return this.properties
}

func (this *Pipeline) GetImports() Imports {
	return this.imports
}

func (this *Pipeline) GetConnections() Connections {
	return this.connections
}
