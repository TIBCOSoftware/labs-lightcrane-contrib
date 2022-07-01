/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package table

import (
	"os"
)

func NewInMenmory(properties map[string]interface{}) *InMemoryTable {
	myTable := &InMemoryTable{
		pKey:          properties["pKey"].([]string),
		indices:       make([][]string, 0),
		theMap:        make(map[CompositKey]*Record),
		theIndexedKey: make(map[CompositKey]map[CompositKey][]CompositKey),
		tableSchema:   properties["tableSchema"].(*Schema),
	}
	indexible := properties["indices"].([]string)
	for index := 0; index < len(indexible); index++ {
		myTable.GenerateKeys(indexible, make([]string, index+1), 0, len(indexible)-1, 0, index+1)
	}
	return myTable
}

type InMemoryTable struct {
	pKey          []string
	indices       [][]string
	tableSchema   *Schema
	theMap        map[CompositKey]*Record
	theIndexedKey map[CompositKey]map[CompositKey][]CompositKey
}

func (this *InMemoryTable) AddIndex(keyName []string) bool {
	key, _ := ConstructKey(keyName, nil)
	this.indices = append(this.indices, keyName)
	this.theIndexedKey[key] = make(map[CompositKey][]CompositKey)
	log.Info("(AddIndex) table after add index: indices = ", this.indices, ", table = ", this.theMap)
	return true
}

func (this *InMemoryTable) RemoveIndex(keyName []string) bool {
	key, _ := ConstructKey(keyName, nil)
	this.theMap[key] = nil
	return true
}

func (this *InMemoryTable) GetPkeyNames() []string {
	return this.pKey
}

func (this *InMemoryTable) GetAll() ([]*Record, bool) {
	records := make([]*Record, 0)
	for _, record := range this.theMap {
		records = append(records, record)
	}
	return records, false
}

func (this *InMemoryTable) Get(searchKey []string, data map[string]interface{}) ([]*Record, bool) {
	log.Info("(Get) searchKey : ", searchKey)
	log.Info("(Get) data : ", data)
	log.Info("(Get) pKey : ", this.pKey)
	log.Info("(Get) theMap : ", this.theMap)

	///////// Need to be fixed ////////
	dumpTable := true
	for _, key := range data {
		if "*" != key.(string) {
			dumpTable = false
			break
		}
	}
	if dumpTable {
		return this.GetAll()
	}
	//////////////////////////////////

	records := make([]*Record, 0)
	pKeyHash, pKeyValueHash := ConstructKey(this.pKey, data)
	searchKeyHash, searchKeyValueHash := ConstructKey(searchKey, data)

	log.Info("(Get) pKeyValueHash : ", pKeyValueHash)
	log.Info("(Get) searchKeyValueHash : ", searchKeyValueHash)

	searchByPKey := true
	if searchKeyHash == pKeyHash {
		log.Info("(Get) Get by primary key !")
		record := this.theMap[pKeyValueHash]
		if nil != record {
			records = append(records, record)
		}
	} else {
		log.Info("(Get) Get by indexed search key !")
		searchByPKey = false
		pKeyValueHashs := this.theIndexedKey[searchKeyHash][searchKeyValueHash]
		if nil != pKeyValueHashs {
			newPKeyValueHashs := make([]CompositKey, 0)
			for _, pKeyValueHash := range pKeyValueHashs {
				if nil != this.theMap[pKeyValueHash] {
					records = append(records, this.theMap[pKeyValueHash])
					newPKeyValueHashs = append(newPKeyValueHashs, pKeyValueHash)
				}
			}
			this.theIndexedKey[searchKeyHash][searchKeyValueHash] = newPKeyValueHashs
		}
	}

	return records, searchByPKey
}

func (this *InMemoryTable) Insert(data map[string]interface{}) bool {
	return false
}

func (this *InMemoryTable) Upsert(data map[string]interface{}) (*Record, *Record) {
	log.Info("(Upsert) data : ", data)
	log.Info("(Upsert) pKey : ", this.pKey)
	log.Info("(Upsert) theMap before : ", this.theMap)

	_, pKeyValueHash := ConstructKey(this.pKey, data)
	record := this.theMap[pKeyValueHash]
	var oldRecord *Record

	if nil != record {
		oldRecord = record.Clone()
		for _, fieldInfo := range *this.tableSchema.DataSchemas() {
			fieldName := fieldInfo["Name"].(string)
			fieldValue := data[fieldName]
			if nil != fieldValue {
				(*record)[fieldName] = fieldValue
			}
		}
	} else {
		// Create new record
		record = &Record{}
		for _, fieldInfo := range *this.tableSchema.DataSchemas() {
			(*record)[fieldInfo["Name"].(string)] = data[fieldInfo["Name"].(string)]
		}
		this.theMap[pKeyValueHash] = record

		// Indexing record
		for _, index := range this.indices {
			indexHash, indexValueHash := ConstructKey(index, data)
			pKeyValueHashs := this.theIndexedKey[indexHash][indexValueHash]
			if nil != pKeyValueHashs {
				this.theIndexedKey[indexHash][indexValueHash] = append(this.theIndexedKey[indexHash][indexValueHash], pKeyValueHash)
			} else {
				this.theIndexedKey[indexHash][indexValueHash] = []CompositKey{pKeyValueHash}
			}
		}
	}

	log.Info("(Upsert) theMap after : ", this.theMap)

	return record, oldRecord
}

func (this *InMemoryTable) Delete(data map[string]interface{}) *Record {
	log.Info("(Delete) data : ", data)
	log.Info("(Delete) pKey : ", this.pKey)
	log.Info("(Delete) theMap before : ", this.theMap)

	_, pKeyValueHash := ConstructKey(this.pKey, data)
	record := this.theMap[pKeyValueHash]

	if nil != record {
		delete(this.theMap, pKeyValueHash)
	}

	log.Info("(Delete) theMap after : ", this.theMap)

	return record
}

func (this *InMemoryTable) RowCount() int {
	return len(this.theMap)
}

func (this *InMemoryTable) Load(file *os.File) {
}

func (this *InMemoryTable) SaveSchema(file *os.File) {
}

func (this *InMemoryTable) SaveData(file *os.File) {
}

func (this *InMemoryTable) GenerateKeys(arr []string, data []string, start int, end int, index int, r int) {
	log.Info("(GenerateKeys) GenerateKeys, index = ", index, ", r = ", r, ", arr", arr)
	if index == r {
		log.Info("(GenerateKeys) GenerateKeys, data = ", data)
		key := make([]string, 0)
		for j := 0; j < r; j++ {
			key = append(key, data[j])
		}
		this.AddIndex(key)
		return
	}

	i := start
	for i <= end && end-i+1 >= r-index {
		data[index] = arr[i]
		this.GenerateKeys(arr, data, i+1, end, index+1, r)
		i += 1
	}
}
