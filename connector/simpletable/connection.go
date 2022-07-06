package simpletable

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/table"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/connection"
	logger "github.com/project-flogo/core/support/log"
)

var log = logger.ChildLogger(logger.RootLogger(), "Simpletable.connection")
var factory = &SimpletableFactory{}

func NewSetting(settings map[string]interface{}) (*Settings, error) {
	s := &Settings{}

	var err = metadata.MapToStruct(settings, s, false)

	if err != nil {
		return nil, err
	}

	if s.Name == "" {
		return nil, errors.New("Required Parameter Name is missing")
	}

	//	Description := s.Description

	if s.Properties == "" {
		return nil, errors.New("Required Properties is missing")
	}

	if s.Schema == "" {
		return nil, errors.New("Required Schema is missing")
	}
	return s, nil
}

// Settings for dgraph
type Settings struct {
	Name        string `md:"name,required"`
	Description string `md:"description"`
	Properties  string `md:"properties,required"`
	Schema      string `md:"schema"`
}

func (s *Settings) ToMap() map[string]interface{} {

	properties := map[string]interface{}{
		"name":        s.Name,
		"description": s.Description,
		"properties":  s.Properties,
		"schema":      s.Schema,
	}

	return properties
}

func init() {
	if os.Getenv(logger.EnvKeyLogLevel) == "DEBUG" {
		log.DebugEnabled()
	}

	err := connection.RegisterManagerFactory(factory)
	if err != nil {
		panic(err)
	}
}

// SimpletableFactory for postgres connection
type SimpletableFactory struct {
}

// Type SimpletableFactory
func (this *SimpletableFactory) Type() string {
	return "Simpletable"
}

// NewManager SimpletableFactory
func (this *SimpletableFactory) NewManager(settings map[string]interface{}) (connection.Manager, error) {

	s, err := NewSetting(settings)

	if err != nil {
		return nil, err
	}

	properties := s.ToMap()
	log.Debug("properties : ", properties)

	sharedConn := &SimpletableManager{
		name:       s.Name,
		properties: properties,
		connection: map[string]table.Table{},
	}

	return sharedConn, nil
}

// SimpletableManager details
type SimpletableManager struct {
	mux        sync.Mutex
	name       string
	properties map[string]interface{}
	connection map[string]table.Table
}

func (this *SimpletableManager) Lookup(clientID string, properties map[string]interface{}) (table.Table, error) {
	var err error
	if nil == this.connection[clientID] {
		this.mux.Lock()
		defer this.mux.Unlock()
		if nil == this.connection[clientID] {
			log.Debug("(getTable) init : ", "initialize table begin ....")

			propertiesArray := []interface{}{}
			var tablename string
			var schema []interface{}
			iSchema := properties["schema"]
			if nil == iSchema {
				return nil, fmt.Errorf("(getTable)Unable to get schema string")
			}
			err := json.Unmarshal([]byte(iSchema.(string)), &schema)
			if nil != err {
				return nil, err
			}
			iProperties := properties["properties"]
			if nil != iProperties {
				err := json.Unmarshal([]byte(iProperties.(string)), &propertiesArray)
				if nil != err {
					return nil, err
				}
			}
			tablename = properties["name"].(string)

			if "" == tablename {
				return nil, fmt.Errorf("(getTable)Unable to get table name")
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

			properties["tableType"] = table.IN_MEMORY
			for _, field := range propertiesArray {
				properties[field.(map[string]interface{})["Name"].(string)] = field.(map[string]interface{})["Value"]
			}

			myTable := table.GetTableManager().GetTable(tablename)
			if nil == myTable {
				tableSchema := table.CreateSchema(&schemaArray)
				properties["pKey"] = keyName
				properties["indices"] = indexible
				properties["tablename"] = tablename
				properties["tableSchema"] = tableSchema
				myTable = table.GetTableManager().CreateTable(properties)
			}
			this.connection[clientID] = myTable
		}
	}
	return this.connection[clientID], err
}

// Type SimpletableManager details
func (this *SimpletableManager) Type() string {
	return "Simpletable"
}

// GetConnection SimpletableManager details
func (this *SimpletableManager) GetConnection() interface{} {
	return this.connection
}

// ReleaseConnection SimpletableManager details
func (this *SimpletableManager) ReleaseConnection(connection interface{}) {

}

// Start SimpletableManager details
func (this *SimpletableManager) Start() error {
	return nil
}

// Stop SimpletableManager details
func (this *SimpletableManager) Stop() error {
	log.Debug("Cleaning up Graph")

	return nil
}