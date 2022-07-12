/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package table

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

func NewRedis(properties map[string]interface{}) (*Redis, error) {
	log.Info("(NewRedis) Entering ........")

	DB := int(properties["DB"].(float64))

	rdb := redis.NewClient(&redis.Options{
		Addr:     properties["Addr"].(string),
		Password: properties["Password"].(string),
		DB:       DB,
	})

	_, err := rdb.Ping(context.Background()).Result()
	if nil != err {
		return nil, err
	}

	// Get a new mutex for synchronizing pipelines.
	mutexname := "default"
	if nil != properties["synchGroupID"] {
		mutexname = properties["synchGroupID"].(string)
	}
	mutexname = fmt.Sprintf("%d-%s", DB, mutexname)
	pool := goredis.NewPool(rdb)
	mutex := redsync.New(pool).NewMutex(mutexname)
	log.Info("(NewRedis) Get mutex for group : ", mutexname)

	return &Redis{
		mutex:       mutex,
		rdb:         rdb,
		pKey:        properties["pKey"].([]string),
		tableSchema: properties["tableSchema"].(*Schema),
	}, nil
}

type Redis struct {
	mutex       *redsync.Mutex
	rdb         *redis.Client
	pKey        []string
	tableSchema *Schema
}

func (this *Redis) AddIndex(keyName []string) bool {
	return true
}

func (this *Redis) RemoveIndex(keyName []string) bool {
	return true
}

func (this *Redis) GetPkeyNames() []string {
	return nil
}

func (this *Redis) GetAll() ([]*Record, bool) {
	return nil, false
}

func (this *Redis) Get(searchKey []string, data map[string]interface{}) ([]*Record, bool) {
	log.Info("(Redis.Get) searchKey : ", searchKey)
	log.Info("(Redis.Get) data : ", data)
	log.Info("(Redis.Get) pKey : ", this.pKey)

	pKeyHash, pKeyValueHash := ConstructKey(this.pKey, data)
	searchKeyHash, searchKeyValueHash := ConstructKey(searchKey, data)

	log.Info("(Redis.Get) pKeyValueHash : ", pKeyValueHash)
	log.Info("(Redis.Get) searchKeyValueHash : ", searchKeyValueHash)

	if searchKeyHash != pKeyHash {
		return make([]*Record, 0), true
	}

	redisKey := strconv.FormatUint(pKeyValueHash.Id, 10)
	log.Info("(Redis.Get) redisKey : ", redisKey)

	ctx := context.Background()
	recordString, err := this.rdb.Get(ctx, redisKey).Result()
	if nil != err || "" == recordString {
		return nil, true
	}
	_, err = this.rdb.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		// record not exits
		return []*Record{}, false
	} else if err != nil {
		// fetch error
		return []*Record{}, false
	}

	var record Record
	if err := json.Unmarshal([]byte(recordString), &record); err != nil {
		log.Error("(Get) Record format incorrect : ", err.Error())
		return nil, true
	}

	records := []*Record{&record}

	return records, true
}

func (this *Redis) Insert(data map[string]interface{}) (*Record, *Record) {
	log.Info("(Redis.Insert) data : ", data)
	log.Info("(Redis.Insert) pKey : ", this.pKey)

	_, pKeyValueHash := ConstructKey(this.pKey, data)
	redisKey := strconv.FormatUint(pKeyValueHash.Id, 10)

	log.Info("(Redis.Insert) pKeyValueHash : ", pKeyValueHash)
	log.Info("(Redis.Insert) redisKey : ", redisKey)

	var oldRecord Record
	var record Record

	// grab lock.
	if err := this.mutex.Lock(); err != nil {
		log.Error("(Redis.Insert) table lock failed.")
	} else {
		log.Info("(Redis.Insert) table locked.")
	}

	ctx := context.Background()
	err := this.rdb.Watch(ctx, func(tx *redis.Tx) error {
		oldRecordString, err := this.rdb.Get(ctx, redisKey).Result()
		if err == redis.Nil {
			// record not exists so we can insert
			recordByte, _ := json.Marshal(data)
			err = this.rdb.Set(ctx, redisKey, string(recordByte), 0).Err()
			if nil != err {
				// Unable to perform insert
				return err
			}
			// record inserted
			record = data
			return nil
		} else if err != nil {
			// prefetch error
			return err
		}

		// record exists
		if err := json.Unmarshal([]byte(oldRecordString), &oldRecord); err != nil {
			log.Error("(Get) Record format incorrect : ", err.Error())
			return err
		}
		return nil

	}, redisKey)

	// Release lock.
	if ok, err := this.mutex.Unlock(); !ok || err != nil {
		log.Error("(Redis.Insert) table unlock failed.")
	} else {
		log.Info("(Redis.Insert) table unlocked.")
	}

	if nil != err {
		log.Error("Error when insert to Redis DB : ", err)
		return nil, nil
	}

	var recordPt *Record
	if 0 < len(record) {
		recordPt = &record
	}
	var oldRecordPt *Record
	if 0 < len(oldRecord) {
		oldRecordPt = &oldRecord
	}

	return recordPt, oldRecordPt
}

func (this *Redis) Upsert(data map[string]interface{}) (*Record, *Record) {
	log.Info("(Redis.Upsert) data : ", data)
	log.Info("(Redis.Upsert) pKey : ", this.pKey)

	_, pKeyValueHash := ConstructKey(this.pKey, data)
	redisKey := strconv.FormatUint(pKeyValueHash.Id, 10)

	log.Info("(Redis.Upsert) pKeyValueHash : ", pKeyValueHash)
	log.Info("(Redis.Upsert) redisKey : ", redisKey)

	var oldRecord Record
	var record Record

	// grab lock.
	if err := this.mutex.Lock(); err != nil {
		log.Error("(Redis.Upsert) table lock failed.")
	} else {
		log.Info("(Redis.Upsert) table locked.")
	}

	ctx := context.Background()
	err := this.rdb.Watch(ctx, func(tx *redis.Tx) error {
		// get old record
		oldRecordString, err := this.rdb.Get(ctx, redisKey).Result()
		if err == redis.Nil {
		} else if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(oldRecordString), &oldRecord); err != nil {
			log.Error("(Get) Record format incorrect : ", err.Error())
			return err
		}

		// set new record
		dataByte, _ := json.Marshal(data)
		err = this.rdb.Set(ctx, redisKey, string(dataByte), 0).Err()
		if nil != err {
			return err
		}

		return nil
	}, redisKey)

	// Release lock.
	if ok, err := this.mutex.Unlock(); !ok || err != nil {
		log.Error("(Redis.Upsert) table unlock failed.")
	} else {
		log.Info("(Redis.Upsert) table unlocked.")
	}

	if nil != err {
		return nil, nil
	} else {
		return &record, &oldRecord
	}
}

func (this *Redis) Delete(data map[string]interface{}) *Record {
	log.Info("(Redis.Delete) data : ", data)

	return nil
}

func (this *Redis) RowCount() int {
	return -1
}
