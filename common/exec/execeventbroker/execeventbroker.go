/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package execeventbroker

import (
	"fmt"
	"sync"
)

var (
	instance *EXEEventBrokerFactory
	once     sync.Once
)

type EXEEventListener interface {
	ProcessEvent(event map[string]interface{}) error
}

type EXEEventBrokerFactory struct {
	exeEventBrokers map[string]*EXEEventBroker
	mux             sync.Mutex
}

func GetFactory() *EXEEventBrokerFactory {
	once.Do(func() {
		instance = &EXEEventBrokerFactory{exeEventBrokers: make(map[string]*EXEEventBroker)}
	})
	return instance
}

func (this *EXEEventBrokerFactory) GetEXEEventBroker(serverId string) *EXEEventBroker {
	return this.exeEventBrokers[serverId]
}

func (this *EXEEventBrokerFactory) CreateEXEEventBroker(
	serverId string,
	listener EXEEventListener) (*EXEEventBroker, error) {

	this.mux.Lock()
	defer this.mux.Unlock()
	broker := this.exeEventBrokers[serverId]

	broker = &EXEEventBroker{
		listener: listener,
	}
	this.exeEventBrokers[serverId] = broker

	return broker, nil
}

type EXEEventBroker struct {
	listener EXEEventListener
}

func (this *EXEEventBroker) Start() {
	fmt.Println("Start broker, EXEEventBroker : ", this)
}

func (this *EXEEventBroker) Stop() {
}

func (this *EXEEventBroker) SendEvent(event map[string]interface{}) {
	this.listener.ProcessEvent(event)
}
