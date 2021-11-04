/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package execlistener

import (
	"context"
	"sync"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/exec/execeventbroker"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

const (
	EVENT_BROKER = "eventBoker"
)

//-============================================-//
//   Entry point register Trigger & factory
//-============================================-//

var triggerMd = trigger.NewMetadata(&Settings{}, &HandlerSettings{}, &Output{})

func init() {
	_ = trigger.Register(&ExecListener{}, &Factory{})
}

//-===============================-//
//     Define Trigger Factory
//-===============================-//

type Factory struct {
}

// Metadata implements trigger.Factory.Metadata
func (*Factory) Metadata() *trigger.Metadata {
	return triggerMd
}

// New implements trigger.Factory.New
func (*Factory) New(config *trigger.Config) (trigger.Trigger, error) {
	settings := &Settings{}
	err := metadata.MapToStruct(config.Settings, settings, true)
	if err != nil {
		return nil, err
	}

	return &ExecListener{settings: settings}, nil
}

//-=========================-//
//      Define Trigger
//-=========================-//

var logger log.Logger

type ExecListener struct {
	metadata *trigger.Metadata
	config   *trigger.Config
	broker   *execeventbroker.EXEEventBroker
	mux      sync.Mutex

	settings *Settings
	handlers []trigger.Handler
}

// implements trigger.Initializable.Initialize
func (this *ExecListener) Initialize(ctx trigger.InitContext) error {

	this.handlers = ctx.GetHandlers()
	logger = ctx.Logger()

	return nil
}

// implements ext.Trigger.Start
func (this *ExecListener) Start() error {

	logger.Info("Start")
	handlers := this.handlers

	logger.Info("Processing handlers")

	for _, handler := range handlers {
		brokerId, exist := handler.Settings()[EVENT_BROKER]
		if !exist {
			logger.Info("EXE event broker is not configured", "TGDB-EXE-4001", nil)
			continue
		}

		var err error
		this.broker, err = execeventbroker.GetFactory().CreateEXEEventBroker(brokerId.(string), this)
		if nil != err {
			return err
		}
		logger.Info("Server = ", *this.broker)
		go this.broker.Start()
	}

	return nil
}

// implements ext.Trigger.Stop
func (this *ExecListener) Stop() error {
	this.broker.Stop()
	return nil
}

func (this *ExecListener) ProcessEvent(event map[string]interface{}) error {
	this.mux.Lock()
	defer this.mux.Unlock()
	logger.Debug("Got Exec event : ", event)
	outputData := &Output{}
	outputData.Event = event
	logger.Debug("Send Exec event out : ", outputData)

	_, err := this.handlers[0].Handle(context.Background(), outputData)
	if nil != err {
		logger.Info("Error -> ", err)
	}

	return err
}
