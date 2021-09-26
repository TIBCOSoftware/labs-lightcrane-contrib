/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package objectbuilder

import (
	"bytes"
	//	"fmt"
	"reflect"
	"strconv"

	//"strings"
	"github.com/TIBCOSoftware/flogo-lib/logger"
)

/* GOLangObjectHandler : for build app */

var log = logger.GetLogger("tibco-labs-lib-objectbuilder")

type FlogoBuilder struct {
}

func (this FlogoBuilder) Build(handler GOLangObjectHandler, object interface{}) interface{} {
	objectWalker := NewGOLangObjectWalker(handler)
	return objectWalker.Start(object)
}

func (this FlogoBuilder) GetData() []map[string]interface{} {
	log.Info("(FlogoBuilder.GetData) Should be overrided!!")
	return nil
}

func (this FlogoBuilder) HandleElements(namespace ElementId, element interface{}, dataType interface{}) interface{} {
	log.Info("(FlogoBuilder.HandleElements) Should be overrided!!")
	return nil
}

/* Build trigger */

func NewFlogoTriggerBuilder() *FlogoTriggerBuilder {
	return &FlogoTriggerBuilder{}
}

type FlogoTriggerBuilder struct {
	FlogoBuilder
}

func (this FlogoTriggerBuilder) HandleElements(namespace ElementId, element interface{}, dataType interface{}) interface{} {
	return nil
}

/* Build activity */

func NewFlogoActivityBuilder() *FlogoActivityBuilder {
	return &FlogoActivityBuilder{}
}

type FlogoActivityBuilder struct {
	FlogoBuilder
}

func (this FlogoActivityBuilder) HandleElements(namespace ElementId, element interface{}, dataType interface{}) interface{} {
	return nil
}

/* Object modifier */

func SetObject(data map[string]interface{}, key string, value interface{}) interface{} {
	builder := NewFlogoAppBuilder(map[string]*Element{
		key: NewElement(key, value, "[]interface {}"),
	})
	return builder.Build(builder, data)
}

func NewFlogoAppBuilder(arrtibuteMap map[string]*Element) *FlogoAppBuilder {
	return &FlogoAppBuilder{
		arrtibuteMap: arrtibuteMap,
	}
}

type FlogoAppBuilder struct {
	FlogoBuilder
	arrtibuteMap map[string]*Element
}

func (this FlogoAppBuilder) HandleElements(namespace ElementId, element interface{}, dataType interface{}) interface{} {
	elementIds := namespace.GetId()
	log.Debug("Handle : id = ", elementIds, ", element = ", element, ", type = ", dataType)
	for _, elementId := range elementIds {
		elementDef := this.arrtibuteMap[elementId]
		log.Debug("map = ", this.arrtibuteMap, ", key = ", elementId, ", value = ", elementDef)
		if nil != elementDef {
			return elementDef.GetDValue()
		}
	}
	return nil
}

/* String Replace Handler */

func StringReplacement(data map[string]interface{}, replacements map[string]interface{}) interface{} {
	replaceHandler := NewStringReplaceHandler(replacements)
	return replaceHandler.Build(replaceHandler, data)
}

func NewStringReplaceHandler(replacements map[string]interface{}) *StringReplaceHandler {
	return &StringReplaceHandler{
		replacements: replacements,
	}
}

type StringReplaceHandler struct {
	FlogoBuilder
	replacements map[string]interface{}
}

func (this StringReplaceHandler) HandleElements(namespace ElementId, element interface{}, dataType interface{}) interface{} {
	if "string" == dataType {
		//log.Info(">>>>>>>> Handle : element = ", element, ", type = ", dataType, ", this.replacements[element = ", this.replacements)
		if nil != this.replacements[element.(string)] {
			//log.Info("XXXXXXXXXXXXXXXXX Handle : element = ", element, ", type = ", dataType, ", this.replacements[elemen = ", this.replacements[element.(string)])
			return this.replacements[element.(string)]
		}
	}
	return nil
}

/* Object Locator */

func LocateObject(data map[string]interface{}, key string) interface{} {
	locator := NewObjectLocator(map[string]*Element{
		key: NewElement(key, nil, "[]interface {}"),
	})
	locator.Build(locator, data)
	return locator.GetResult()[key].GetDValue()
}

func NewObjectLocator(arrtibuteMap map[string]*Element) *ObjectLocator {
	return &ObjectLocator{
		arrtibuteMap: arrtibuteMap,
	}
}

type ObjectLocator struct {
	FlogoBuilder
	arrtibuteMap map[string]*Element
}

func (this ObjectLocator) HandleElements(namespace ElementId, element interface{}, dataType interface{}) interface{} {
	elementIds := namespace.GetId()
	log.Debug("Handle : id = ", elementIds, ", element = ", element, ", type = ", dataType)
	for _, elementId := range elementIds {
		elementDef := this.arrtibuteMap[elementId]
		log.Debug("map = ", this.arrtibuteMap, ", key = ", elementId, ", value = ", elementDef)
		if nil != elementDef {
			elementDef.SetDValue(element)
			return nil
		}
	}
	return nil
}

func (this ObjectLocator) GetResult() map[string]*Element {
	return this.arrtibuteMap
}

/* GOLang Object Processing Framework */

type Scope struct {
	index    int
	maxIndex int
	name     string
	array    bool
}

type ElementId struct {
	namespace []Scope
	name      interface{}
}

func (this *ElementId) GetIndex() int {
	return this.namespace[len(this.namespace)-1].index
}

func (this *ElementId) GetId() []string {
	ids := make([]string, 1)

	var buffer bytes.Buffer
	arrayElement := false
	for i := range this.namespace {
		if !arrayElement {
			if 0 != i {
				buffer.WriteString(".")
			}
			buffer.WriteString(this.namespace[i].name)
		} else {
			arrayElement = false
		}

		if this.namespace[i].array {
			buffer.WriteString("[")
			if -1 < this.namespace[i].index {
				buffer.WriteString(strconv.Itoa(this.namespace[i].index))
			}
			buffer.WriteString("]")
			arrayElement = true
		}
	}
	if nil != this.name {
		buffer.WriteString(".")
		buffer.WriteString(this.name.(string))
	}
	ids[0] = buffer.String()
	return ids
}

func (this *ElementId) SetName(name string) {
	this.name = name
}

func (this *ElementId) updateIndex(index int, maxIndex int) {
	log.Debug("   Before updateIndex : ", this.namespace, ", index : ", index)
	this.namespace[len(this.namespace)-1].index = index
	this.namespace[len(this.namespace)-1].maxIndex = maxIndex
	log.Debug("   After updateIndex : ", this.namespace, ", index : ", index)
}

func (this *ElementId) enterScope(scopename string, isArray bool) {
	log.Debug("Before enterScope : ", this.namespace, ", ID = ", this.GetId()) //, ", index : ", this.namespace[len(this.namespace)-1].index)
	this.name = nil
	this.namespace = append(this.namespace, Scope{name: scopename, array: isArray, index: -1, maxIndex: -1})
	log.Debug("After enterScope : ", this.namespace, ", ID = ", this.GetId()) //, ", index : ", this.namespace[len(this.namespace)-1].index)
}

func (this *ElementId) leaveScope(scopename string, isArray bool) {
	log.Debug("Before leaveScope : ", this.namespace, ", ID = ", this.GetId()) //, ", index : ", this.namespace[len(this.namespace)-1].index)
	this.namespace = this.namespace[:len(this.namespace)-1]
	this.name = nil
	log.Debug("After leaveScope : ", this.namespace, ", ID = ", this.GetId()) //, ", index : ", this.namespace[len(this.namespace)-1].index)
}

/** Element **/

func NewElement(name string, dValue interface{}, dataType string) *Element {
	return &Element{
		name:     name,
		dValue:   dValue,
		dataType: dataType,
	}
}

type Element struct {
	name     string
	dValue   interface{}
	dataType string
}

func (this *Element) SetName(name string) {
	this.name = name
}

func (this *Element) GetName() string {
	return this.name
}

func (this *Element) SetDValue(dValue interface{}) {
	this.dValue = dValue
}

func (this *Element) GetDValue() interface{} {
	return this.dValue
}

func (this *Element) SetType(dataType string) {
	this.dataType = dataType
}

func (this *Element) GetType() string {
	return this.dataType
}

/* GOLangObjectHandler interface */

type GOLangObjectHandler interface {
	HandleElements(namespace ElementId, element interface{}, dataType interface{}) interface{}
	GetData() []map[string]interface{}
}

/* GOLangObjectWalker class */

type GOLangObjectWalker struct {
	ElementId
	currentLevel  int
	objectHandler GOLangObjectHandler
}

func NewGOLangObjectWalker(objectHandler GOLangObjectHandler) GOLangObjectWalker {
	GOLangObjectWalker := GOLangObjectWalker{
		currentLevel:  0,
		objectHandler: objectHandler}
	GOLangObjectWalker.ElementId = ElementId{
		namespace: make([]Scope, 0),
	}

	return GOLangObjectWalker
}

func (this *GOLangObjectWalker) Start(objectData interface{}) interface{} {
	log.Debug("%%%%%%%", objectData)
	this.walk("root", objectData)
	return objectData
}

func (this *GOLangObjectWalker) walk(name string, data interface{}) interface{} {

	var modifiedData interface{}
	switch data.(type) {
	case []interface{}:
		{
			this.startArray(name)
			modifiedData = this.objectHandler.HandleElements(this.ElementId, data, "[]interface{}")
			if nil == modifiedData {
				dataArray := data.([]interface{})
				maxIndex := len(dataArray) - 1
				for index, subdata := range dataArray {
					this.updateIndex(index, maxIndex)
					log.Debug("=====>", name, " ===>", subdata)
					this.walk(name, subdata)
				}
				this.updateIndex(-1, -1)
			}
			this.endArray(name)
			break
		}
	case map[string]interface{}:
		{
			this.startObject(name)
			modifiedData = this.objectHandler.HandleElements(this.ElementId, data, "map[string]interface{}")
			if nil == modifiedData {
				modifiedMap := make(map[string]interface{})
				dataMap := data.(map[string]interface{})
				for subname, subdata := range dataMap {
					modifiedSubdata := this.walk(subname, subdata)
					if nil != modifiedSubdata {
						modifiedMap[subname] = modifiedSubdata
					}
				}
				for subname, subdata := range modifiedMap {
					dataMap[subname] = subdata
				}
			}
			this.endObject(name)
			break
		}
	default:
		{
			this.ElementId.SetName(name)
			log.Debug("Got element -> ", name, ", --> ", data, ", --> ", reflect.TypeOf(data).String())
			modifiedData = this.objectHandler.HandleElements(this.ElementId, data, reflect.TypeOf(data).String())
		}
	}
	return modifiedData
}

func (this *GOLangObjectWalker) startArray(name string) {
	log.Debug("Start Array before scope -> ", name, ", ", this.namespace)
	this.ElementId.enterScope(name, true)
	log.Debug("Start Array after scope -> ", name, ", ", this.namespace)
}

func (this *GOLangObjectWalker) endArray(name string) {
	log.Debug("End Array -> ", name)
	this.ElementId.leaveScope(name, true)
}

func (this *GOLangObjectWalker) startObject(name string) {
	this.ElementId.enterScope(name, false)
	log.Debug("Start Object -> ", name, ", ", this.namespace)
}

func (this *GOLangObjectWalker) endObject(name string) {
	log.Debug("End Object -> ", name)
	this.ElementId.leaveScope(name, false)
}
