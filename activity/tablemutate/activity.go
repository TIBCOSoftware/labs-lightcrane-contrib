/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package tablemutate

import (
	"reflect"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/table"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/connector/simpletable"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/metadata"
	logger "github.com/project-flogo/core/support/log"
)

var log = logger.ChildLogger(logger.RootLogger(), "activity-table-mutate")

const (
	setting_Table  = "Table"
	setting_Method = "Method"
	Method_Insert  = "insert"
	Method_Upsert  = "upsert"
	Method_Delete  = "delete"
	output_Data    = "Data"
	output_Exists  = "Exists"
)

var activityMd = activity.ToMetadata(&Input{}, &Output{})

func init() {

	_ = activity.Register(&Activity{}, New)
}

// TableMutateActivity is an Activity that is used to Filter a message to the console
type Activity struct {
	settings *Settings
	tableMgr *simpletable.SimpletableManager
}

// New creates a new AppActivity
func New(ctx activity.InitContext) (activity.Activity, error) {
	settings := &Settings{}
	err := metadata.MapToStruct(ctx.Settings(), settings, true)
	if err != nil {
		return nil, err
	}

	tableMgr := settings.Table.(*simpletable.SimpletableManager)
	act := &Activity{
		settings: settings,
		tableMgr: tableMgr,
	}
	return act, nil
}

// Metadata returns the activity's metadata
func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

// Eval implements api.Activity.Eval - Filters the Message
func (a *Activity) Eval(ctx activity.Context) (done bool, err error) {

	log.Debug("(TableMutateActivity.Eval) entering ..... ")
	defer log.Debug("(TableMutateActivity.Eval) exit ..... ")

	input := &Input{}
	err = ctx.GetInputObject(input)
	if err != nil {
		return false, err
	}

	myTable, err := a.tableMgr.Lookup(ctx.Name(), a.settings.ToMap())

	if nil != err {
		return false, err
	}

	iData := input.Mapping

	log.Debug("(Eval)iData.Value = ", iData, ", type = ", reflect.TypeOf(iData))

	method := a.settings.Method
	var newRecord *table.Record
	var oldRecord *table.Record
	switch method {
	case Method_Insert:
		newRecord, oldRecord = myTable.Insert(iData)
	case Method_Upsert:
		newRecord, oldRecord = myTable.Upsert(iData)
	case Method_Delete:
		oldRecord = myTable.Delete(iData)
	}
	exists := oldRecord != nil

	outputTuple := map[string]interface{}{
		"New": newRecord,
		"Old": oldRecord,
	}

	log.Debug("(Eval)output tuple = ", outputTuple, ", exist = ", exists, ", row count = ", myTable.RowCount())

	ctx.SetOutput(output_Data, outputTuple)
	ctx.SetOutput(output_Exists, exists)

	return true, nil
}
