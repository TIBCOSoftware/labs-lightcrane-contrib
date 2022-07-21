/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package fileserver

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

//-============================================-//
//   Entry point register Trigger & factory
//-============================================-//

var triggerMd = trigger.NewMetadata(&Settings{}, &HandlerSettings{}, &Output{})

func init() {
	_ = trigger.Register(&TcpServer{}, &Factory{})
}

//-===============================-//
//     Define Trigger Factory
//-===============================-//

type Factory struct {
}

func (*Factory) Metadata() *trigger.Metadata {
	return triggerMd
}

func (*Factory) New(config *trigger.Config) (trigger.Trigger, error) {
	settings := &Settings{}
	err := metadata.MapToStruct(config.Settings, settings, true)
	if err != nil {
		return nil, err
	}

	return &TcpServer{settings: settings}, nil
}

//-=========================-//
//      Define Trigger
//-=========================-//

var logger log.Logger

type TcpServer struct {
	metadata *trigger.Metadata
	config   *trigger.Config
	mux      sync.Mutex

	settings *Settings
	handlers []trigger.Handler
}

func (this *TcpServer) Initialize(ctx trigger.InitContext) error {
	this.handlers = ctx.GetHandlers()
	logger = ctx.Logger()

	return nil
}

func (this *TcpServer) Start() error {
	logger.Info("(Start) Processing handlers new ....")
	for _, handler := range this.handlers {
		handlerSetting := &HandlerSettings{}
		err := metadata.MapToStruct(handler.Settings(), handlerSetting, true)
		if err != nil {
			return err
		}

		http.Handle(handlerSetting.URLPath, http.StripPrefix(handlerSetting.URLPath, http.FileServer(http.Dir(handlerSetting.Dir))))
		logger.Info("(Start) Started URLPath = ", handlerSetting.URLPath, ", Dir = ", handlerSetting.Dir, ", port = ", this.settings.Port)
		logger.Infof("Serving %s on HTTP port: %s\n", handlerSetting.Dir, this.settings.Port)

		if this.settings.EnableTLS {
			err = http.ListenAndServeTLS(fmt.Sprintf(":%s", this.settings.Port), this.settings.CertFile, this.settings.KeyFile, nil)
		} else {
			err = http.ListenAndServe(fmt.Sprintf(":%s", this.settings.Port), nil)
		}

		if err != nil {
			logger.Error("Error happen while calling ListenAndServe: ", err)
		}
	}
	logger.Info("(Start) Now started")

	return nil
}

func (this *TcpServer) Stop() error {
	return nil
}
