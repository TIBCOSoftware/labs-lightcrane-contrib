package table

import (
	"testing"
)

const (
	iPorts      = "ports"
	iProperties = "properties"
)

func TestInMemoryUpsert(t *testing.T) {
	log.Info("(TestInMemoryUpsert) Entering ......")
	defer log.Info("(TestInMemoryUpsert) exit ......")

	schemaArray := []map[string]interface{}{
		map[string]interface{}{
			"Name":    "ProjectID",
			"Type":    "String",
			"IsKey":   "yes",
			"Indexed": "no",
		},
		map[string]interface{}{
			"Name":    "Name",
			"Type":    "String",
			"IsKey":   "yes",
			"Indexed": "no",
		},
		map[string]interface{}{
			"Name":    "Data",
			"Type":    "Object",
			"IsKey":   "no",
			"Indexed": "no",
		},
	}
	tableSchema := CreateSchema(&schemaArray)
	properties := map[string]interface{}{
		"pKey":        []string{"ProjectID"},
		"indices":     []string{},
		"tabletype":   IN_MEMORY,
		"tablename":   "inMemory",
		"tableSchema": tableSchema,
	}
	myTable, _ := GetTableManager().CreateTable(properties)

	record, oldRecord := myTable.Upsert(map[string]interface{}{
		"ProjectID": "1234567890",
		"Name":      "Test Project",
		"Data": map[string]interface{}{
			"aaa": 1,
			"bbb": "bbb",
		},
	})

	log.Info("record = ", record, ", oldRecord = ", oldRecord)
}

func TestRedisInsert(t *testing.T) {
	log.Info("(TestRedisInsert) Entering ......")
	defer log.Info("(TestRedisInsert) exit ......")

	schemaArray := []map[string]interface{}{
		map[string]interface{}{
			"Name":    "ProjectID",
			"Type":    "String",
			"IsKey":   "yes",
			"Indexed": "no",
		},
		map[string]interface{}{
			"Name":    "Name",
			"Type":    "String",
			"IsKey":   "yes",
			"Indexed": "no",
		},
		map[string]interface{}{
			"Name":    "Data",
			"Type":    "Object",
			"IsKey":   "no",
			"Indexed": "no",
		},
	}
	tableSchema := CreateSchema(&schemaArray)
	properties := map[string]interface{}{
		"pKey":        []string{"ProjectID"},
		"tableType":   REDIS,
		"tablename":   "redis",
		"tableSchema": tableSchema,
		"Addr":        "192.168.1.152:6379",
		"Password":    "eYVX7EwVmmxKPCDmwMtyKVge8oLd2t81",
		"DB":          float64(0),
	}
	myTable, _ := GetTableManager().CreateTable(properties)

	record, oldRecord := myTable.Insert(map[string]interface{}{
		"ProjectID": "1234567890z",
		"Name":      "Test Project",
		"Data": map[string]interface{}{
			"aaa": 1,
			"bbb": "bbb",
		},
	})

	log.Info("record = ", record, ", oldRecord = ", oldRecord)
}
