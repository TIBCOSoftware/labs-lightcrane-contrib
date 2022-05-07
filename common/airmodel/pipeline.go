/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */

package airmodel

import (
	"encoding/json"
	"fmt"

	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/objectbuilder"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
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
	normalFlows     []Logic
	errorFlows      []Logic
	notifiers       map[string]Notifier
	listeners       map[string]interface{}
	contributes     Contributes
	properties      Properties
	connections     Connections
	imports         Imports
}

func (this *Pipeline) Build2() (string, error) {
	log.Info("(Pipeline.Build2) Build data source : ", this.normalFlows[0].GetCategory())
	flogoFlows := make([]interface{}, 0)
	this.dataSource.Build(fmt.Sprintf("%s_%d", this.normalFlows[0].GetCategory(), 0))
	this.contributes.Add(this.dataSource.GetContribution())
	this.imports.Add(this.dataSource.GetImports())
	this.properties.Add(this.dataSource.GetID(), this.dataSource.GetProperties(), this.dataSource.GetRawProperties(), nil)
	this.connections.Add(this.dataSource.GetConnections())
	flogoFlows = append(flogoFlows, this.dataSource.GetResource())

	normalFlogoFlows, _ := this.buildFlow(this.normalFlows)
	flogoFlows = append(flogoFlows, normalFlogoFlows...)
	errorFlogoFlows, _ := this.buildFlow(this.errorFlows)
	flogoFlows = append(flogoFlows, errorFlogoFlows...)

	/*
		Now we add notifier (flogo trigger) for each listener
	*/
	triggers := this.dataSource.GetTriggers()
	for ID, notifier := range this.notifiers {
		for _, trigger := range notifier.GetTriggers(ID, this.listeners) {
			triggers = append(triggers, trigger)
			this.imports.Add(notifier.GetImports())
		}
	}

	elementMap := make(map[string]*objectbuilder.Element)

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
		//log.Debug("Unable to serialize object, reason : ", err.Error())
		return "", err
	}
	return string(jsondata), err
}

func (this *Pipeline) buildFlow(flows []Logic) ([]interface{}, error) {
	flogoFlows := make([]interface{}, 0)
	for index, logic := range flows {
		log.Info("(Pipeline.buildFlow) Build flow : ", logic.GetCategory(), ", index : ", index)
		if index < len(flows)-1 {
			logic.Build(flows[index+1].GetID(), false)
		} else {
			logic.Build("NA", true)
		}
		this.contributes.Add(logic.GetContribution())
		this.imports.Add(logic.GetImports())

		/*
			Here we turn-on the App.IsListener flag for the logic component (flow)
			which we defined as notification listener in extra block
		*/
		isListener := false
		for _, listenerGroup := range this.listeners {
			log.Info("(Pipeline.buildFlow)  listenerGroup = ", listenerGroup)
			for _, listener := range listenerGroup.([]interface{}) {
				log.Info("(Pipeline.buildFlow)    listener = ", listener)
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
	log.Info("(Pipeline.buildFlow) flogoFlows : ", flogoFlows)
	return flogoFlows, nil
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

		/*
			Here we turn-on the App.IsListener flag for the logic component (flow)
			which we defined as notification listener in extra block
		*/
		isListener := false
		for _, listenerGroup := range this.listeners {
			log.Debug("(Pipeline.Build)  listenerGroup = ", listenerGroup)
			for _, listener := range listenerGroup.([]interface{}) {
				log.Debug("(Pipeline.Build)    listener = ", listener)
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

	/*
		Now we add notifier (flogo trigger) for each listener
	*/
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
		//log.Debug("Unable to serialize object, reason : ", err.Error())
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

func (this *Pipeline) AddNormalLogic(logic Logic) {
	this.normalFlows = append(this.normalFlows, logic)
}

func (this *Pipeline) AddErrorLogic(logic Logic) {
	this.errorFlows = append(this.errorFlows, logic)
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
