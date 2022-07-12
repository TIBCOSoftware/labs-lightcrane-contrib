/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package tablequery

import (
	"reflect"
	"strings"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/connector/simpletable"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/metadata"
	logger "github.com/project-flogo/core/support/log"
)

// activityLogger is the default logger for the Filter Activity
var log = logger.ChildLogger(logger.RootLogger(), "activity-table-query")

const (
	setting_Table   = "Table"
	setting_Indices = "Indices"
	input           = "QueryKey"
	output_Data     = "Data"
	output_Exists   = "Exists"
)

var activityMd = activity.ToMetadata(&Input{}, &Output{})

func init() {
	_ = activity.Register(&Activity{}, New)
}

// TableMutateActivity is an Activity that is used to Filter a message to the console
type Activity struct {
	settings  *Settings
	queryKeys []string
	tableMgr  *simpletable.SimpletableManager
}

// New creates a new AppActivity
func New(ctx activity.InitContext) (activity.Activity, error) {
	settings := &Settings{}
	err := metadata.MapToStruct(ctx.Settings(), settings, true)
	if err != nil {
		return nil, err
	}

	queryKeys := strings.Split(settings.Indices, " ")

	tableMgr := settings.Table.(*simpletable.SimpletableManager)
	act := &Activity{
		queryKeys: queryKeys,
		settings:  settings,
		tableMgr:  tableMgr,
	}
	return act, nil
}

// Metadata returns the activity's metadata
func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

// Eval implements api.Activity.Eval - Filters the Message
func (a *Activity) Eval(ctx activity.Context) (done bool, err error) {

	log.Debug("(TableQueryActivity.Eval) entering ..... ")
	defer log.Debug("(TableQueryActivity.Eval) exit ..... ")

	input := &Input{}
	err = ctx.GetInputObject(input)
	if err != nil {
		return false, err
	}

	myTable, err := a.tableMgr.Lookup(ctx.Name(), a.settings.ToMap())

	if nil != err {
		return false, err
	}

	iData := input.QueryKey
	log.Debug("(Eval)iData.Value = ", iData, ", type = ", reflect.TypeOf(iData))

	records, byPKey := myTable.Get(a.queryKeys, iData)
	exists := (len(records) != 0)

	log.Debug("(Eval)output tuple = ", records, ", exist = ", exists, ", found row count = ", len(records))

	if byPKey {
		if exists {
			ctx.SetOutput(output_Data, records[0])
		} else {
			ctx.SetOutput(output_Data, nil)
		}

	} else {
		ctx.SetOutput(output_Data, records)
	}
	ctx.SetOutput(output_Exists, exists)

	return true, nil
}
