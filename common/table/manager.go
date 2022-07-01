/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package table

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"sync"

	"github.com/TIBCOSoftware/flogo-lib/logger"
)

var log = logger.GetLogger("tibco-f1-table")

const (
	REDIS     = "Redis"
	IN_MEMORY = "InMemory"
)

type TableManager struct {
	tables map[string]Table
}

var (
	instance *TableManager
	once     sync.Once
	mux      sync.Mutex
)

func GetTableManager() *TableManager {
	once.Do(func() {
		instance = &TableManager{tables: make(map[string]Table)}
	})
	return instance
}

func (this *TableManager) GetTable(tablename string) Table {
	return this.tables[tablename]
}

func (this *TableManager) CreateTable(properties map[string]interface{}) Table {

	tablename := properties["tablename"].(string)
	tableType := ""
	if nil != properties["tableType"] {
		tableType = properties["tableType"].(string)
	}
	table := this.tables[tablename]
	if nil == table {
		mux.Lock()
		defer mux.Unlock()
		table = this.tables[tablename]
		if nil == table {
			log.Info("Table type : ", tableType)
			if REDIS == tableType {
				table = NewRedis(properties)
			} else {
				table = NewInMenmory(properties)
			}
			this.tables[tablename] = table
		}
	}

	return table
}

type Table interface {
	AddIndex(keyName []string) bool
	RemoveIndex(keyName []string) bool
	GetPkeyNames() []string
	GetAll() ([]*Record, bool)
	Get(searchKey []string, data map[string]interface{}) ([]*Record, bool)
	Insert(data map[string]interface{}) (*Record, *Record)
	Upsert(data map[string]interface{}) (*Record, *Record)
	Delete(data map[string]interface{}) *Record
	RowCount() int
}

type Record map[string]interface{}

func (this *Record) Clone() *Record {
	record := &Record{}
	for key, value := range *this {
		(*record)[key] = value
	}
	return record
}

func CreateSchema(schema *[]map[string]interface{}) *Schema {
	return &Schema{
		schema: schema,
	}
}

type Schema struct {
	schema *[](map[string]interface{})
}

func (this *Schema) DataSchemas() *[](map[string]interface{}) {
	return this.schema
}

func (this *Schema) Length() int {
	return len(*this.schema)
}

type CompositKey struct {
	Id uint64
}

func KeyFromDataArray(elements []interface{}) CompositKey {
	keyBytes := []byte{}
	for _, element := range elements {
		elementBytes, _ := json.Marshal(element)
		keyBytes = append(keyBytes, elementBytes...)
	}
	hasher := md5.New()
	hasher.Write(keyBytes)
	return CompositKey{Id: binary.BigEndian.Uint64(hasher.Sum(nil))}
}

func ConstructKey(keyNameStrs []string, tuple map[string]interface{}) (CompositKey, CompositKey) {
	log.Debug("(ConstructKey) keyNameStrs : ", keyNameStrs)
	log.Debug("(ConstructKey) tuple : ", tuple)

	/* build key */
	key := make([]interface{}, len(keyNameStrs))
	keyFields := make([]interface{}, len(keyNameStrs))
	for j, keyNameStr := range keyNameStrs {
		key[j] = tuple[keyNameStr]
		keyFields[j] = keyNameStr
	}
	log.Debug("(ConstructKey) keyFields : ", keyFields)
	log.Debug("(ConstructKey) key : ", key)

	return KeyFromDataArray(keyFields), KeyFromDataArray(key)
}
