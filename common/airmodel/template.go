/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */

package airmodel

import (
	"io/ioutil"

	"fmt"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/objectbuilder"
)

/* FlogoTemplateLibrary */

type FlogoTemplateLibrary struct {
	pipeline            Pipeline
	components          map[string]map[string]PipelineComponent
	componentDescriptor map[string]interface{}
}

func (this *FlogoTemplateLibrary) GetPipeline() Pipeline {
	return this.pipeline.Clone()
}

func (this *FlogoTemplateLibrary) GetComponent(sn int, category string, name string, properties []interface{}) PipelineComponent {
	if nil == this.components[category][name] {
		return nil
	}
	component := this.components[category][name].Clone(sn, name, properties)
	component.SetRuntimeProperties(properties)
	return component
}

func (this *FlogoTemplateLibrary) GetComponentDescriptor(category string, name string) interface{} {
	log.Debugf("[FlogoTemplateLibrary) GetComponentDescriptor - category = %s", category, ", component = %s ", name)
	if "*" == category {
		if "*" == name {
			// category : *, name : * -> all components' properties
			return this.componentDescriptor
		} else {
			// category : *, name : not * -> all category name
			categotires := make([]interface{}, len(this.componentDescriptor))
			index := 0
			for categoryName, _ := range this.componentDescriptor {
				categotires[index] = categoryName
				index++
			}
			return map[string]interface{}{
				"Categories": categotires,
			}
		}
	} else {
		components := this.componentDescriptor[category].(map[string]interface{})
		if "*" == name {
			// category : not *, name : * -> all category's component
			return components
		} else if "" != name {
			// category : not *, name : not empty -> all component's properties
			return components[name]
		} else {
			// category : not *, name : not empty & not * -> all component's properties
			componentNamess := make([]interface{}, len(components))
			index := 0
			for compoenetName, _ := range components {
				componentNamess[index] = compoenetName
				index++
			}
			return map[string]interface{}{
				"Components": componentNamess,
			}
		}
	}
	return nil
}

func NewFlogoTemplateLibrary(folder string) (*FlogoTemplateLibrary, error) {
	//log.Debug(folder)
	pipeline, err := NewPipeline("AirPipeline", fmt.Sprintf("%s/pipeline.json", folder))
	if nil != err {
		return nil, err
	}
	components := make(map[string]map[string]PipelineComponent)
	componentDescriptor := make(map[string]interface{})

	categories, _ := ioutil.ReadDir(folder)
	for _, category := range categories {
		if category.IsDir() {
			log.Debug("Category : ", category.Name())
			templates, _ := ioutil.ReadDir(folder + "/" + category.Name())
			if nil == components[category.Name()] {
				components[category.Name()] = make(map[string]PipelineComponent)
			}
			subflowFileMain := fmt.Sprintf("%s/SubflowEntryMain.json", folder)
			subflowDataMain, err := FromFile(subflowFileMain)
			if nil != err {
				log.Error("Fail to read main subflow data from %s : %v", subflowFileMain, err)
			}
			log.Debug("subflowDataMain : ", subflowDataMain)
			subflowEntryMain := objectbuilder.LocateObject(subflowDataMain, "root.resources[0].data.tasks[0]").(map[string]interface{})
			subflowFile := fmt.Sprintf("%s/SubflowEntry.json", folder)
			subflowData, err := FromFile(subflowFile)
			if nil != err {
				log.Error("Fail to read subflow data from %s : %v", subflowFile, err)
			}
			subflowEntry := objectbuilder.LocateObject(subflowData, "root.resources[0].data.tasks[0]").(map[string]interface{})

			metadataData, err := FromFile(fmt.Sprintf("%s/Metadata.json", folder))
			if nil != err {
				log.Error("Fail to read subflow metadata from %s : %v", fmt.Sprintf("%s/Metadata.json", folder), err)
			}
			subflowMetadata := objectbuilder.LocateObject(metadataData, "root.resources[0].data.metadata").(map[string]interface{})

			errorHandlerData, err := FromFile(fmt.Sprintf("%s/ErrorHandler.json", folder))
			if nil != err {
				log.Error("Fail to read subflow errorHandler from %s : %v", fmt.Sprintf("%s/ErrorHandler.json", folder), err)
			}
			subflowErrorHandler := objectbuilder.LocateObject(errorHandlerData, "root.resources[0].data.errorHandler").(map[string]interface{})

			for _, template := range templates {
				if template.IsDir() {
					log.Debug("---- template -> " + template.Name())
					filename := fmt.Sprintf("%s/%s/%s/%s.json", folder, category.Name(), template.Name(), template.Name())
					var component PipelineComponent
					if "DataSource" == category.Name() {
						component, err = NewDataSource(category.Name(), filename, subflowEntryMain)
					} else if "Notifier" == category.Name() {
						component, err = NewNotifier(category.Name(), filename)
					} else {
						component, err = NewLogic(category.Name(), filename, subflowEntry, subflowMetadata, subflowErrorHandler)
					}
					if nil != err {
						return nil, err
					}
					components[category.Name()][template.Name()] = component
					//log.Debugf("---- component.GetData() -> %v", component.GetData())
				}
			}
		}
	}

	for categoryName, category := range components {
		categoryComponents := componentDescriptor[categoryName]
		if nil == categoryComponents {
			categoryComponents = make(map[string]interface{})
			componentDescriptor[categoryName] = categoryComponents
		}
		for componentName, component := range category {
			categoryComponents.(map[string]interface{})[componentName] = component.GetProperties()
		}
	}

	return &FlogoTemplateLibrary{
		pipeline:            pipeline,
		components:          components,
		componentDescriptor: componentDescriptor,
	}, nil
}

/* PipelineComponent Interface */

type PipelineComponent interface {
	GetData() map[string]interface{}
	GetProperties() []interface{}
	GetRuntimeProperties() []interface{}
	SetRuntimeProperties(runtimeProperties []interface{})
	Clone(sn int, name string, runtimeProperties []interface{}) PipelineComponent
}

/* BasePipelineComponent BaseClass */

type BasePipelineComponent struct {
	data map[string]interface{}
}

func (this BasePipelineComponent) GetData() map[string]interface{} {
	return this.data
}

func (this BasePipelineComponent) Get(key string) interface{} {
	if nil != this.data[key] {
		return this.data[key]
	}
	return make([]interface{}, 0)
}

func (this BasePipelineComponent) Set(key string, value interface{}) interface{} {
	if nil != this.data[key] {
		return this.data[key]
	}
	return make([]interface{}, 0)
}
