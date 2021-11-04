/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package httpredirect

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

const (
	cPort = "Port"
	cPath = "Path"
)

//-============================================-//
//   Entry point register Trigger & factory
//-============================================-//

var triggerMd = trigger.NewMetadata(&Settings{}, &HandlerSettings{}, &Output{})

func init() {
	_ = trigger.Register(&HTTPRedirect{}, &Factory{})
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

	return &HTTPRedirect{settings: settings}, nil
}

//-=========================-//
//      Define Trigger
//-=========================-//

var logger log.Logger

type HTTPRedirect struct {
	metadata *trigger.Metadata
	config   *trigger.Config
	mux      sync.Mutex

	settings *Settings
	handlers []trigger.Handler
}

// implements trigger.Initializable.Initialize
func (this *HTTPRedirect) Initialize(ctx trigger.InitContext) error {
	this.handlers = ctx.GetHandlers()
	logger = ctx.Logger()

	return nil
}

// implements ext.Trigger.Start
func (this *HTTPRedirect) Start() error {
	logger.Info("(Start) Processing handlers")
	for _, handler := range this.handlers {
		handlerSetting := &HandlerSettings{}
		err := metadata.MapToStruct(handler.Settings(), handlerSetting, true)
		if err != nil {
			return err
		}

		go func() {
			http.HandleFunc(handlerSetting.Path, this.redirect(&handler))
			err = http.ListenAndServe(fmt.Sprintf(":%s", this.settings.Port), nil)
			if err != nil {
				logger.Error("ListenAndServe: ", err)
			}
		}()
		logger.Info("(Start) Started path = ", handlerSetting.Path, ", port = ", this.settings.Port)
	}
	logger.Info("(Start) Now started")

	return nil
}

func (this *HTTPRedirect) redirect(handler *trigger.Handler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("(Serve) Request URL : ", r.URL)

		outputData := make(map[string]interface{})
		outputData["RequestURL"] = r.URL.Path

		gContext := context.Background()
		results, err := (*handler).Handle(gContext, outputData)
		if nil != err {
			logger.Error(err)
		}
		logger.Info(results)

		http.Redirect(w, r, results["data"].(map[string]interface{})["RedirectURL"].(string), 307)

	}
}

// implements ext.Trigger.Stop
func (this *HTTPRedirect) Stop() error {
	return nil
}
