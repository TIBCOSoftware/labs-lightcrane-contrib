/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package mapping

import (
	"sync"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

// activityLogger is the default logger for the Filter Activity
var log = logger.GetLogger("labs-lc-activity-mapping")

const (
	input          = "Mapping"
	is_array       = "IsArray"
	skip_condition = "SkipCondition"
	array_size     = "ArraySize"
	output         = "Data"
)

// Mapping is an Activity that is used to Filter a message to the console
type Mapping struct {
	metadata     *activity.Metadata
	initialized  bool
	mappedTuples map[string]*ProcessedList
	mux          sync.Mutex
}

// NewActivity creates a new AppActivity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	aCSVParserActivity := &Mapping{
		metadata:     metadata,
		mappedTuples: make(map[string]*ProcessedList),
	}
	return aCSVParserActivity
}

// Metadata returns the activity's metadata
func (a *Mapping) Metadata() *activity.Metadata {
	return a.metadata
}

// Eval implements api.Activity.Eval - Filters the Message
func (a *Mapping) Eval(ctx activity.Context) (done bool, err error) {

	log.Debug("[Mapping:Eval] entering ........ ")
	defer log.Debug("[Mapping:Eval] exit ........ ")

	mappedTuple := ctx.GetInput(input).(map[string]interface{})
	log.Debug("[Mapping.Evale] mapped data = ", mappedTuple)

	skipCondition := false
	if nil != mappedTuple[skip_condition] {
		skipCondition = mappedTuple[skip_condition].(bool)
	}
	isArray, exists := ctx.GetSetting(is_array)
	if exists && isArray.(bool) {
		mappedTuples := a.getMappedTuples(ctx)
		log.Debug("[Mapping.Evale] skipCondition = ", skipCondition)
		if !skipCondition {
			delete(mappedTuple, array_size)
			delete(mappedTuple, skip_condition)
			mappedTuples.SetData(mappedTuple)
			log.Debug("[Mapping.Evale] 1. mappedTuples.GetList() = ", mappedTuples.GetList())
		} else {
			mappedTuples.SkipData()
			log.Debug("[Mapping.Evale] 2. mappedTuples.GetList() = ", mappedTuples.GetList())
		}
		arraySize := mappedTuple[array_size].(int)
		if arraySize == mappedTuples.ProcessedCount() {
			log.Debug("[Mapping.Evale] 3. mappedTuples.GetList() = ", mappedTuples.GetList())
			ctx.SetOutput(output, mappedTuples.GetList())
			mappedTuples.clear()
		}
	} else {
		if !skipCondition {
			log.Debug("[Mapping.Evale] mappedTuple = ", mappedTuple)
			ctx.SetOutput(output, mappedTuple)
		}
	}
	return true, nil
}

func (a *Mapping) getMappedTuples(context activity.Context) *ProcessedList {
	myId := util.ActivityId(context)
	mappedTuples := a.mappedTuples[myId]
	if nil == mappedTuples {
		mappedTuples = a.mappedTuples[myId]
		if nil == mappedTuples {
			mappedTuples = &ProcessedList{
				dataArray:      make([]interface{}, 0),
				processedCount: 0,
			}
			a.mappedTuples[myId] = mappedTuples
		}
	}

	return mappedTuples
}

type ProcessedList struct {
	processedCount int
	dataArray      []interface{}
}

func (this *ProcessedList) SetData(data interface{}) {
	this.dataArray = append(this.dataArray, data)
	this.processedCount += 1
}

func (this *ProcessedList) SkipData() {
	this.processedCount += 1
}

func (this *ProcessedList) GetList() []interface{} {
	return this.dataArray
}

func (this *ProcessedList) ProcessedCount() int {
	return this.processedCount
}

func (this *ProcessedList) Length() int {
	return len(this.dataArray)
}

func (this *ProcessedList) clear() {
	this.dataArray = make([]interface{}, 0)
	this.processedCount = 0
}
