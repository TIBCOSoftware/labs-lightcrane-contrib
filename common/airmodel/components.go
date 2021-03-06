/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */

package airmodel

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	kwr "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/keywordreplace"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/objectbuilder"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

/* DataSource Class */

func NewDataSource(category string, datasource string, subflowActivity map[string]interface{}) (DataSource, error) {
	data, err := FromFile(datasource)
	return DataSource{
		category:          category,
		data:              data,
		rawProperties:     data["properties"].([]interface{}),
		defaultActivities: objectbuilder.LocateObject(data, "root.resources[0].data.tasks[]").([]interface{}),
		subflowActivity:   subflowActivity,
	}, err
}

type DataSource struct {
	category          string
	name              string
	data              map[string]interface{}
	rawProperties     []interface{}
	runtimeProperties []interface{}
	defaultActivities []interface{}
	subflowActivity   map[string]interface{}
}

func (this DataSource) GetRuntimeProperties() []interface{} {
	return this.runtimeProperties
}

func (this DataSource) SetRuntimeProperties(runtimeProperties []interface{}) {
	this.runtimeProperties = runtimeProperties
}

func (this DataSource) addNamespace4Properties(ID string) {
	log.Debug(">>>>>>>>>> Data source >>>>>>>>>>>>> Data = ", this.data)
	handler := ObjectStringValueReplaceHandler{
		ID: fmt.Sprintf("%s_", ID),
	}
	handler.Build(handler, this.data)
}

func (this DataSource) Build(subflowID string) {

	activities := this.BuildActivities(subflowID)
	links := objectbuilder.LocateObject(this.data, "root.resources[0].data.links[]").([]interface{})
	if 0 == len(links) {
		for index, _ := range activities {
			if 0 != index {
				links = append(links, map[string]interface{}{
					"id":   index,
					"from": activities[index-1].(map[string]interface{})["id"],
					"to":   activities[index].(map[string]interface{})["id"],
					"type": "default",
				})
			}
		}
		log.Debug(">>>>>>>>>> links >>>>>>>>>>>>> links = ", links)
		_ = objectbuilder.SetObject(this.data, "root.resources[0].data.links[]", links)
	}

	_ = objectbuilder.SetObject(this.data, "root.resources[0].data.tasks[]", activities)
	_ = objectbuilder.SetObject(
		this.data, "root.resources[0].data.tasks[0].activity.input.message",
		fmt.Sprintf("=string.concat(\"########## DataSource ##########\", coerce.toString($flow.data))"))
	this.addNamespace4Properties(this.category)
}

func (this DataSource) BuildActivities(subflowID string) []interface{} {
	log.Info("------->", this.name)
	activities := make([]interface{}, 0)
	for index, activity := range this.defaultActivities {
		if 0 < index { //} == len(this.defaultActivities)-1 {
			previousActivityId := this.defaultActivities[index-1].(map[string]interface{})["id"].(string)
			log.Info("$$$$$$$$$$$$$$$$$$$", previousActivityId)
			if strings.HasPrefix(previousActivityId, "Next_Flow") {
				log.Info(">>>>>>>>>>>>>>>>>>>>>Found Next_Flow", previousActivityId)
				subflowActivity := this.defaultActivities[index-1].(map[string]interface{})
				_ = objectbuilder.SetObject(subflowActivity, "root.activity.settings.flowURI", fmt.Sprintf("res://flow:%s", subflowID))
			} else if strings.HasPrefix(previousActivityId, "NewFlowData_") {
				log.Info(">>>>>>>>>>>>>>>>>>>>>Found NewFlowData_", previousActivityId)
				branchId := "Next_Flow_" + previousActivityId[len("NewFlowData_"):]
				subflowActivity := util.DeepCopy(this.subflowActivity).(map[string]interface{})
				_ = objectbuilder.SetObject(subflowActivity, "root.id", branchId)
				_ = objectbuilder.SetObject(subflowActivity, "root.name", branchId)
				_ = objectbuilder.SetObject(subflowActivity, "root.settings.iterate", fmt.Sprintf("=$activity[%s].Data.readings", previousActivityId))
				_ = objectbuilder.SetObject(subflowActivity, "root.activity.input.gateway", fmt.Sprintf("=$activity[%s].Data.gateway", previousActivityId))
				_ = objectbuilder.SetObject(subflowActivity, "root.activity.input.reading", "=$iteration[value]")
				_ = objectbuilder.SetObject(subflowActivity, "root.activity.input.enriched", fmt.Sprintf("=$activity[%s].Data.enriched", previousActivityId))
				_ = objectbuilder.SetObject(subflowActivity, "root.activity.settings.flowURI", fmt.Sprintf("res://flow:%s", subflowID))
				activities = append(activities, subflowActivity)
			}
			log.Info(">>>>>>>>>>>>>>>>>>>>>length", len(activities))
		}
		if index == len(this.defaultActivities)-1 &&
			index == len(activities) {
			subflowActivity := util.DeepCopy(this.subflowActivity).(map[string]interface{})
			_ = objectbuilder.SetObject(subflowActivity, "root.activity.settings.flowURI", fmt.Sprintf("res://flow:%s", subflowID))
			activities = append(activities, subflowActivity)
		}

		activities = append(activities, activity)
		log.Info(">>>>>>>>>>>>>>>>>>>>>length", len(activities))
	}
	return activities
}

func (this DataSource) GetID() string {
	return this.category
}

func (this DataSource) GetData() map[string]interface{} {
	return this.data
}

func (this DataSource) Clone(sn int, name string, runtimeProperties []interface{}) PipelineComponent {
	return DataSource{
		category:          this.category,
		name:              name,
		data:              util.DeepCopy(this.data).(map[string]interface{}),
		rawProperties:     util.DeepCopy(this.rawProperties).([]interface{}),
		runtimeProperties: runtimeProperties,
		defaultActivities: util.DeepCopy(this.defaultActivities).([]interface{}),
		subflowActivity:   util.DeepCopy(this.subflowActivity).(map[string]interface{}),
	}
}

func (this DataSource) Get(key string) interface{} {
	if nil != this.data[key] {
		return this.data[key]
	}
	return make([]interface{}, 0)
}

func (this DataSource) GetTriggers() []interface{} {
	//return this.triggers
	return this.data["triggers"].([]interface{})
}

func (this DataSource) GetResource() interface{} {
	return this.data["resources"].([]interface{})[0]
}

func (this DataSource) GetContribution() interface{} {
	return this.data["contrib"]
}

func (this DataSource) GetImports() []interface{} {
	return this.data["imports"].([]interface{})
}

func (this DataSource) GetRawProperties() []interface{} {
	return this.rawProperties
}

func (this DataSource) GetProperties() []interface{} {
	return this.data["properties"].([]interface{})
}

func (this DataSource) GetConnections() interface{} {
	return this.data["connections"]
}

/* Notifier Class */

func NewNotifier(category string, datasource string) (Notifier, error) {
	data, err := FromFile(datasource)
	return Notifier{
		category:      category,
		data:          data,
		rawProperties: data["properties"].([]interface{}),
	}, err
}

type Notifier struct {
	category          string
	name              string
	data              map[string]interface{}
	rawProperties     []interface{}
	runtimeProperties []interface{}
}

func (this Notifier) GetRuntimeProperties() []interface{} {
	return this.runtimeProperties
}

func (this Notifier) SetRuntimeProperties(runtimeProperties []interface{}) {
	this.runtimeProperties = runtimeProperties
}

func (this Notifier) addNamespace4Properties(ID string) {
	log.Debug(">>>>>>>>>> Notifier >>>>>>>>>>>>> Data = ", this.data)
}

func (this Notifier) Build(subflowID string) {

}

func (this Notifier) BuildTriggers(subflowID string) []interface{} {
	triggers := make([]interface{}, 0)
	return triggers
}

func (this Notifier) GetID() string {
	return this.category
}

func (this Notifier) GetData() map[string]interface{} {
	return this.data
}

func (this Notifier) Clone(sn int, name string, runtimeProperties []interface{}) PipelineComponent {
	return Notifier{
		category:      this.category,
		name:          name,
		data:          util.DeepCopy(this.data).(map[string]interface{}),
		rawProperties: util.DeepCopy(this.rawProperties).([]interface{}),
	}
}

func (this Notifier) Get(key string) interface{} {
	if nil != this.data[key] {
		return this.data[key]
	}
	return make([]interface{}, 0)
}

func (this Notifier) GetTriggers(notifierID string, listeners map[string]interface{}) []interface{} {
	log.Debug("(Notifier.GetTriggers) ========== notifierID ->", notifierID)
	log.Debug("(Notifier.GetTriggers) ========== listeners ->", listeners)
	triggers := util.DeepCopy(this.data["triggers"]).([]interface{})
	for _, trigger := range triggers {
		if nil != listeners[notifierID] {
			id := fmt.Sprintf("notificationListener_%s", notifierID)
			_ = objectbuilder.SetObject(trigger.(map[string]interface{}), "root.id", id)
			_ = objectbuilder.SetObject(trigger.(map[string]interface{}), "root.name", id)
			handler := trigger.(map[string]interface{})["handlers"].([]interface{})[0]
			handlers := make([]interface{}, 0)
			for _, listener := range listeners[notifierID].([]interface{}) {
				newHandler := util.DeepCopy(handler).(map[string]interface{})
				_ = objectbuilder.SetObject(newHandler, "root.settings.notifierID", notifierID)
				_ = objectbuilder.SetObject(newHandler, "root.name", listener)
				_ = objectbuilder.SetObject(newHandler, "root.action.settings.flowURI", fmt.Sprintf("res://flow:%s", listener))
				handlers = append(handlers, newHandler)
			}
			trigger.(map[string]interface{})["handlers"] = handlers
		}
	}
	return triggers
}

func (this Notifier) GetContribution() interface{} {
	return this.data["contrib"]
}

func (this Notifier) GetImports() []interface{} {
	return this.data["imports"].([]interface{})
}

func (this Notifier) GetRawProperties() []interface{} {
	return this.rawProperties
}

func (this Notifier) GetProperties() []interface{} {
	return this.data["properties"].([]interface{})
}

func (this Notifier) GetConnections() interface{} {
	return this.data["connections"]
}

/* Logic Class */

func NewLogic(
	category string,
	filename string,
	subflowActivity map[string]interface{},
	metadata map[string]interface{},
	errorHandler map[string]interface{}) (Logic, error) {

	log.Debug(">>>>>>>>>> Logics >>>>>>>>>>>>> category = ", category, ", filename = ", filename)
	log.Debug("metadata = ", metadata)
	log.Debug("errorHandler = ", errorHandler)
	data, err := FromFile(filename)
	iDefaultActivities := objectbuilder.LocateObject(data, "root.resources[0].data.tasks[]")
	var defaultActivities []interface{}
	if nil != iDefaultActivities {
		defaultActivities = iDefaultActivities.([]interface{})
	}

	/////////////////////////////////////////////////////////////////////////
	subflowActivities := make(map[string]interface{})
	subflowPosDef := objectbuilder.LocateObject(data, "root.resources[0].data.description")
	//log.Debug(">>>>>>>>>> Logics >>>>>>>>>>>>> subflowPosDef = ", subflowPosDef)
	if nil != subflowPosDef && "" != subflowPosDef {
		subflowPosStrArray := strings.Split(subflowPosDef.(string), "|")
		//log.Debug(">>>>>>>>>> Logics >>>>>>>>>>>>> subflowPosStrArray = ", subflowPosStrArray)
		for _, posStr := range subflowPosStrArray {
			//log.Debug(">>>>>>>>>> Logics >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> posStr = ", posStr)
			_, err := strconv.Atoi(posStr)
			if nil != err {
				return Logic{}, err
			}
			subflowActivity = util.DeepCopy(subflowActivity).(map[string]interface{})
			_ = objectbuilder.SetObject(subflowActivity, "root.id", fmt.Sprintf("Next_Flow_%s", posStr))
			_ = objectbuilder.SetObject(subflowActivity, "root.name", fmt.Sprintf("Next_Flow_%s", posStr))
			subflowActivities[posStr] = subflowActivity
		}
	} else {
		subflowActivities[strconv.Itoa(len(defaultActivities)-1)] = util.DeepCopy(subflowActivity).(map[string]interface{})
	}
	//log.Debug(">>>>>>>>>> Logics >>>>>>>>>>>>> subflowActivities = ", subflowActivities)
	/////////////////////////////////////////////////////////////////////

	rawProperties := []interface{}{}
	if nil == data["properties"] {
		log.Warn("No properties defined in ", filename)
	} else {
		rawProperties = data["properties"].([]interface{})
	}

	return Logic{
		category:          category,
		data:              data,
		rawProperties:     rawProperties,
		defaultActivities: defaultActivities,
		subflowActivities: subflowActivities,
		metadata:          metadata,
		errorHandler:      errorHandler,
	}, err
}

type Logic struct {
	sn                int
	category          string
	name              string
	data              map[string]interface{}
	rawProperties     []interface{}
	runtimeProperties []interface{}
	defaultActivities []interface{}
	subflowActivities map[string]interface{}
	metadata          map[string]interface{}
	errorHandler      map[string]interface{}
	loggersetup       string
}

func (this Logic) GetID() string {
	return fmt.Sprintf("%s_%d", this.category, this.sn)
}

func (this Logic) GetCategory() string {
	return this.category
}

func (this Logic) GetRuntimeProperties() []interface{} {
	return this.runtimeProperties
}

func (this Logic) SetRuntimeProperties(runtimeProperties []interface{}) {
	this.runtimeProperties = runtimeProperties
}

func (this Logic) addNamespace4Properties(ID string) {
	//log.Debug(">>>>>>>>>> Logics >>>>>>>>>>>>> Data = ", this.data)
	handler := ObjectStringValueReplaceHandler{
		ID: fmt.Sprintf("%s_", ID),
	}
	handler.Build(handler, this.data)
}

func (this Logic) Build(nextSubflowID string, last bool) {
	log.Info("(Build.Build) name = ", this.name, ", nextSubflowID = ", nextSubflowID, ", last = ", last)
	var activities []interface{}
	if !last {
		activities = make([]interface{}, len(this.defaultActivities)+len(this.subflowActivities))
		index := 0
		for _, activity := range this.defaultActivities {
			subflowActivities := this.subflowActivities[strconv.Itoa(index)]
			//log.Debug("$$$$$$$$$$$$$$$$$$$ index = ", index, ", subflowActivities = ", subflowActivities)
			if nil != subflowActivities {
				activities[index] = subflowActivities
				if 0 < index {
					previousActivityId := this.defaultActivities[index-1].(map[string]interface{})["id"].(string)
					//log.Debug("$$$$$$$$$$$$$$$$$$$", previousActivityId)
					if strings.HasPrefix(previousActivityId, "DataEmbedder") {
						_ = objectbuilder.SetObject(subflowActivities.(map[string]interface{}), "root.activity.input.enriched", "=$activity[DataEmbedder].OutputDataCollection")
					} else if strings.HasPrefix(previousActivityId, "NewFlowData") {
						_ = objectbuilder.SetObject(subflowActivities.(map[string]interface{}), "root.activity.input.gateway", "=$activity[NewFlowData].Data.gateway")
						_ = objectbuilder.SetObject(subflowActivities.(map[string]interface{}), "root.activity.input.reading", "=$activity[NewFlowData].Data.reading")
						_ = objectbuilder.SetObject(subflowActivities.(map[string]interface{}), "root.activity.input.enriched", "=$activity[NewFlowData].Data.enriched")
					}
				}
				_ = objectbuilder.SetObject(subflowActivities.(map[string]interface{}), "root.activity.settings.flowURI", fmt.Sprintf("res://flow:%s", nextSubflowID))
				index++
			}
			activities[index] = activity
			index++
		}
		_ = objectbuilder.SetObject(this.data, "root.resources[0].data.tasks[]", activities)
	} else {
		activities = this.defaultActivities
	}

	//log.Debug("$$$$$$$$$$$$$$$$$$$ this.data02 = ", this.data["resources"].([]interface{})[0].(map[string]interface{})["data"])
	links := objectbuilder.LocateObject(this.data, "root.resources[0].data.links[]").([]interface{})
	if 0 == len(links) {
		links := make([]interface{}, len(activities)-1)
		for index, _ := range activities {
			//log.Debug(activities[index])
			if 0 != index {
				links[index-1] = map[string]interface{}{
					"id":   index,
					"from": activities[index-1].(map[string]interface{})["id"],
					"to":   activities[index].(map[string]interface{})["id"],
					"type": "default",
				}
			}
		}
		_ = objectbuilder.SetObject(this.data, "root.resources[0].data.links[]", links)
	}
	_ = objectbuilder.SetObject(this.data, "root.resources[0].data.metadata", this.metadata)
	_ = objectbuilder.SetObject(this.data, "root.resources[0].data.errorHandler", this.errorHandler)
	_ = objectbuilder.SetObject(this.data, "root.resources[0].data.name", fmt.Sprintf("%s_%d", this.category, this.sn))
	_ = objectbuilder.SetObject(this.data, "root.resources[0].id", fmt.Sprintf("flow:%s_%d", this.category, this.sn))
	if "Dummy" != this.name {
		_ = objectbuilder.SetObject(
			this.data, "root.resources[0].data.tasks[0].activity.input.message",
			//fmt.Sprintf("=string.concat(\"########## %s_%d(%s) ########## : gateway = \", $flow.gateway, \", reading = \", coerce.toString($flow.reading), \", enriched = \", coerce.toString($flow.enriched))", this.category, this.sn, this.name))
			fmt.Sprintf("=string.concat(\"########## %s_%d(%s) ########## : gateway = \", $flow.gateway, \", reading = { ... }, enriched = \", coerce.toString($flow.enriched))", this.category, this.sn, this.name))
	}
	this.addNamespace4Properties(fmt.Sprintf("%s_%d", this.category, this.sn))
	log.Info("(Build.Build) nextSubflowID = ", nextSubflowID, ", this.data = ", this.data)
}

func (this Logic) GetData() map[string]interface{} {
	return this.data
}

func (this Logic) Clone(sn int, name string, runtimeProperties []interface{}) PipelineComponent {
	return Logic{
		sn:                sn,
		category:          this.category,
		name:              name,
		runtimeProperties: runtimeProperties,
		data:              util.DeepCopy(this.data).(map[string]interface{}),
		rawProperties:     util.DeepCopy(this.rawProperties).([]interface{}),
		defaultActivities: util.DeepCopy(this.defaultActivities).([]interface{}),
		subflowActivities: util.DeepCopy(this.subflowActivities).(map[string]interface{}),
		metadata:          util.DeepCopy(this.metadata).(map[string]interface{}),
		errorHandler:      util.DeepCopy(this.errorHandler).(map[string]interface{}),
	}
}

func (this Logic) GetRunner() interface{} {
	return this.data["runner"]
}

func (this Logic) GetNotificationBroker() interface{} {
	return this.data["notificationBroker"]
}

func (this Logic) Get(key string) interface{} {
	if nil != this.data[key] {
		return this.data[key]
	}
	return make([]interface{}, 0)
}

func (this Logic) GetResource() interface{} {
	return this.data["resources"].([]interface{})[0]
}

func (this Logic) GetContribution() interface{} {
	return this.data["contrib"]
}

func (this Logic) GetImports() []interface{} {
	return this.data["imports"].([]interface{})
}

func (this Logic) GetRawProperties() []interface{} {
	return this.rawProperties
}

func (this Logic) GetProperties() []interface{} {
	return this.data["properties"].([]interface{})
}

func (this Logic) GetConnections() interface{} {
	return this.data["connections"]
}

func FromFile(filename string) (map[string]interface{}, error) {
	//log.Debug(":::::::::", filename)
	fileContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	json.Unmarshal(fileContent, &result)

	if nil != err {
		return nil, err
	}

	//log.Debug("[BasePipelineComponent:buildFromFile] FlogoTemplate : filename = ", filename, ", template = ", result)
	return result, nil
}

/* Handler properties parameter replacement */

type ObjectStringValueReplaceHandler struct {
	objectbuilder.FlogoBuilder
	ID string
}

func (this ObjectStringValueReplaceHandler) HandleElements(namespace objectbuilder.ElementId, element interface{}, dataType interface{}) interface{} {
	if "string" == dataType {
		log.Debug("(ObjectStringValueReplaceHandler HandleElements) Handle : element = ", element, ", type = ", dataType)
		replacement := kwr.Replace(element.(string), "${{", "}}$", "ID", this.ID)
		if replacement != element {
			log.Debug("(ObjectStringValueReplaceHandler HandleElements) Handle : element = ", element, ", type = ", dataType, ", replacement = ", replacement)
			return replacement
		}
	}
	return nil
}
