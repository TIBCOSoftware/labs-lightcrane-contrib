/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package tableupsert

import (
	"encoding/json"
	"reflect"
	"sync"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/table"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

// activityLogger is the default logger for the Filter Activity
var log = logger.GetLogger("activity-table-upsert")

const (
	setting_Table   = "Table"
	setting_Indices = "Indices"
	input           = "Mapping"
	output_Data     = "Data"
	output_Exists   = "Exists"
)

// TableUpsertActivity is an Activity that is used to Filter a message to the console
type TableUpsertActivity struct {
	metadata        *activity.Metadata
	activityToTable map[string]string
	mux             sync.Mutex
}

// NewActivity creates a new AppActivity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	aTableActivity := &TableUpsertActivity{
		metadata:        metadata,
		activityToTable: make(map[string]string),
	}
	return aTableActivity
}

// Metadata returns the activity's metadata
func (a *TableUpsertActivity) Metadata() *activity.Metadata {
	return a.metadata
}

// Eval implements api.Activity.Eval - Filters the Message
func (a *TableUpsertActivity) Eval(ctx activity.Context) (done bool, err error) {

	log.Debug("(TableUpsertActivity.Eval) entering ..... ")
	defer log.Debug("(TableUpsertActivity.Eval) exit ..... ")

	myTable, err := a.getTable(ctx)

	if nil != err {
		return false, err
	}

	iData := ctx.GetInput(input).(*data.ComplexObject).Value

	log.Debug("(Eval)iData.Value = ", iData, ", type = ", reflect.TypeOf(iData))

	outputTuple, _ := myTable.Upsert(iData.(map[string]interface{}))
	exists := outputTuple != nil

	log.Debug("(Eval)output tuple = ", outputTuple, ", exist = ", exists, ", row count = ", myTable.RowCount())

	complexdata := &data.ComplexObject{Metadata: "Data", Value: outputTuple}
	ctx.SetOutput(output_Data, complexdata)
	ctx.SetOutput(output_Exists, exists)

	return true, nil
}

func (a *TableUpsertActivity) getTable(context activity.Context) (table.Table, error) {
	myId := util.ActivityId(context)

	myTable := table.GetTableManager().GetTable(a.activityToTable[myId])
	if nil == myTable {
		a.mux.Lock()
		defer a.mux.Unlock()

		myTable = table.GetTableManager().GetTable(a.activityToTable[myId])
		if nil == myTable {

			log.Debug("(getTable) init : ", "initialize table begin ....")

			iTableInfo, exist := context.GetSetting(setting_Table)
			if !exist {
				return nil, activity.NewError("(getTable)Table is not configured", "TABLE_UPSERT-4002", nil)
			}

			//Read table details
			tableInfo, _ := data.CoerceToObject(iTableInfo)
			if tableInfo == nil {
				return nil, activity.NewError("(getTable)Unable extract table details", "TABLE_UPSERT-4001", nil)
			}

			tabletype := table.IN_MEMORY
			propertiesArray := []interface{}{}
			var tablename string
			var schema []interface{}
			tableSettings, _ := tableInfo["settings"].([]interface{})
			if tableSettings != nil {
				for _, v := range tableSettings {
					setting, _ := data.CoerceToObject(v)

					if nil != setting {
						if setting["name"] == "schema" {
							iSchema := setting["value"]
							if nil == iSchema {
								return nil, activity.NewError("(getTable)Unable to get schema string", "TABLE_UPSERT-4004", nil)
							}
							err := json.Unmarshal([]byte(iSchema.(string)), &schema)
							if nil != err {
								return nil, err
							}
						} else if setting["name"] == "Properties" {
							iProperties := setting["value"]
							if nil != iProperties {
								err := json.Unmarshal([]byte(iProperties.(string)), &propertiesArray)
								if nil != err {
									return nil, err
								}
							}
						} else if setting["name"] == "type" {
							if nil != setting["value"] {
								tabletype = setting["value"].(string)
							}
						} else if setting["name"] == "name" {
							tablename = setting["value"].(string)
						}
					}
				}
			}

			if "" == tablename {
				return nil, activity.NewError("(getTable)Unable to get table name", "TABLE_UPSERT-4003", nil)
			}

			log.Debug("-============= TABLE SCHEMA ================-")
			log.Debug(schema)
			log.Debug("-===========================================-")
			log.Debug("-============= TABLE PROPERTIES ================-")
			log.Debug(propertiesArray)
			log.Debug("-===============================================-")

			properties := make(map[string]interface{})
			keyName := make([]string, 0)
			indexible := make([]string, 0)
			schemaArray := make([](map[string]interface{}), len(schema))
			for index, field := range schema {
				schemaArray[index] = field.(map[string]interface{})
				if "yes" == schemaArray[index]["IsKey"].(string) {
					keyName = append(keyName, schemaArray[index]["Name"].(string))
					indexible = append(indexible, schemaArray[index]["Name"].(string))
				}
				if "yes" == schemaArray[index]["Indexed"].(string) {
					indexible = append(indexible, schemaArray[index]["Name"].(string))
				}
			}

			for _, field := range propertiesArray {
				properties[field.(map[string]interface{})["Name"].(string)] = field.(map[string]interface{})["Value"]
			}

			myTable = table.GetTableManager().GetTable(tablename)
			if nil == myTable {
				tableSchema := table.CreateSchema(&schemaArray)
				properties["pKey"] = keyName
				properties["indices"] = indexible
				properties["tableType"] = tabletype
				properties["tablename"] = tablename
				properties["tableSchema"] = tableSchema
				myTable = table.GetTableManager().CreateTable(properties)
			}

			log.Debug("(getTable) init : ", "initialize table done : myTable = ", myTable)
			a.activityToTable[myId] = tablename
		}
	}

	return myTable, nil
}
