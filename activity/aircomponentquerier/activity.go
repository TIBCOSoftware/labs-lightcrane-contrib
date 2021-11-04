/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */

/*
	{
		"imports": [],
		"name": "ProjectAirApplication",
		"description": "",
		"version": "1.0.0",
		"type": "flogo:app",
		"appModel": "1.1.1",
		"feVersion": "2.9.0",
		"triggers": [],
		"resources": [],
		"properties": [],
		"connections": {},
		"contrib": "",
		"fe_metadata": ""
	}
*/

package aircomponentquerier

import (
	"sync"

	model "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/airmodel"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
)

var log = logger.GetLogger("tibco-f1-aircomponentquerier")

var initialized bool = false

const (
	sTemplateFolder = "TemplateFolder"
	iCategory       = "Category"
	iComponent      = "Component"
	oDescriptor     = "Descriptor"
)

type PipelineBuilderActivity struct {
	metadata  *activity.Metadata
	mux       sync.Mutex
	templates map[string]*model.FlogoTemplateLibrary
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aPipelineBuilderActivity := &PipelineBuilderActivity{
		metadata:  metadata,
		templates: make(map[string]*model.FlogoTemplateLibrary),
	}

	return aPipelineBuilderActivity
}

func (a *PipelineBuilderActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *PipelineBuilderActivity) Eval(context activity.Context) (done bool, err error) {

	log.Debug("[PipelineBuilderActivity:Eval] entering ........ ")
	defer log.Debug("[PipelineBuilderActivity:Eval] Exit ........ ")

	templateLibrary, err := a.getTemplateLibrary(context)
	if err != nil {
		return false, err
	}

	category := context.GetInput(iCategory)
	if nil == category {
		category = "*"
	}
	log.Debug("[PipelineBuilderActivity:Eval]  Name : ", category)

	component := context.GetInput(iComponent)
	if nil == component {
		component = ""
	}
	log.Debug("[PipelineBuilderActivity:Eval]  Name : ", component)

	descriptor := templateLibrary.GetComponentDescriptor(category.(string), component.(string))

	log.Debug("[PipelineBuilderActivity:Eval]  oDescriptor : ", descriptor)
	context.SetOutput(oDescriptor, descriptor)

	return true, nil
}

func (a *PipelineBuilderActivity) getTemplateLibrary(ctx activity.Context) (*model.FlogoTemplateLibrary, error) {

	log.Debug("[PipelineBuilderActivity:getTemplate] entering ........ ")
	defer log.Debug("[PipelineBuilderActivity:getTemplate] exit ........ ")

	myId := util.ActivityId(ctx)
	templateLib := a.templates[myId]

	if nil == templateLib {
		a.mux.Lock()
		defer a.mux.Unlock()
		templateLib = a.templates[myId]
		if nil == templateLib {
			templateFolderSetting, exist := ctx.GetSetting(sTemplateFolder)
			if !exist {
				return nil, activity.NewError("Template is not configured", "PipelineBuilder-4002", nil)
			}
			templateFolder := templateFolderSetting.(string)
			var err error
			templateLib, err = model.NewFlogoTemplateLibrary(templateFolder)
			if nil != err {
				return nil, err
			}
			a.templates[myId] = templateLib
		}
	}
	return templateLib, nil
}
