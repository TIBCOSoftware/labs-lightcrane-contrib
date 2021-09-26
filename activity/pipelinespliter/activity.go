/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package pipelinespliter

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/project-flogo/core/app"
	"github.com/project-flogo/core/app/resource"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/data/schema"
	"github.com/project-flogo/core/engine/secret"
	"github.com/project-flogo/core/trigger"
	"github.com/project-flogo/flow/definition"
)

var log = logger.GetLogger("tibco-model-ops-pipelinespliter")

var initialized bool = false

const (
	sTemplate              = "Template"
	iPipelineConfig        = "RawPipelineConfig"
	oID                    = "ID"
	oDataFlow              = "DataFlow"
	oPipelineConfig        = "PipelineConfig"
	ComponentType_External = "ext:app"
	ComponentType_Flogo    = "flogo:app"
)

type PipelineSpliterActivity struct {
	metadata *activity.Metadata
	mux      sync.Mutex
	spliters map[string]*PipelineConfigSpliter
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aPipelineSpliterActivity := &PipelineSpliterActivity{
		metadata: metadata,
		spliters: make(map[string]*PipelineConfigSpliter),
	}

	return aPipelineSpliterActivity
}

func (a *PipelineSpliterActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *PipelineSpliterActivity) Eval(context activity.Context) (done bool, err error) {

	log.Info("[PipelineSpliterActivity:Eval] entering ........ ")

	spliter, err := a.getSpliter(context)
	if err != nil {
		return false, err
	}

	rawPipelineConfig, ok := context.GetInput(iPipelineConfig).(string)
	if !ok {
		return false, errors.New("Invalid command ... ")
	}

	//log.Info("rawPipelineConfig : ", rawPipelineConfig)

	rawPipelineConfigNoSecret, err := secret.PreProcessConfig([]byte(rawPipelineConfig))
	if err != nil {
		return false, err
	}

	appConfig := &PipelineConfig{}
	err = json.Unmarshal(rawPipelineConfigNoSecret, &appConfig)
	if err != nil {
		return false, err
	}

	fmt.Println("========================================")
	deployment, err := spliter.Split(appConfig)
	fmt.Println("========================================")
	if nil != err {
		return false, err
	}

	pipelineConfigsOutput := make([]interface{}, len(deployment.PipelineConfigs))
	for index, pipelineConfig := range deployment.PipelineConfigs {
		componentConfig, _ := json.Marshal(pipelineConfig)
		pipelineConfigsOutput[index] = map[string]interface{}{
			"Name":            pipelineConfig.Name,
			"Type":            pipelineConfig.Type,
			"ComponentConfig": string(componentConfig),
			"Properties":      pipelineConfig.RunnerProperties,
		}
	}

	//log.Info(pipelineConfigsOutput)

	context.SetOutput(oID, deployment.ID)
	context.SetOutput(oDataFlow, &data.ComplexObject{Metadata: oDataFlow, Value: deployment.DataFlow})
	context.SetOutput(oPipelineConfig, &data.ComplexObject{Metadata: oPipelineConfig, Value: pipelineConfigsOutput})

	log.Info("[PipelineSpliterActivity:Eval] Exit ........ ")

	return true, nil
}

func (a *PipelineSpliterActivity) getSpliter(ctx activity.Context) (*PipelineConfigSpliter, error) {

	log.Info("[PipelineSpliterActivity:getSpliter] entering ........ ")
	defer log.Info("[PipelineSpliterActivity:getSpliter] exit ........ ")

	myId := ActivityId(ctx)
	spliter := a.spliters[myId]

	if nil == spliter {
		a.mux.Lock()
		defer a.mux.Unlock()
		spliter = a.spliters[myId]
		if nil == spliter {
			templateSetting, exist := ctx.GetSetting(sTemplate)
			if !exist {
				return nil, activity.NewError("Template is not configured", "PipelineSpliter-4002", nil)
			}
			var err error
			templateObj, err := data.CoerceToObject(templateSetting)
			if err != nil {
				return nil, err
			}
			templateString, err := b64.StdEncoding.DecodeString(strings.Split(templateObj["content"].(string), ",")[1])
			if err != nil {
				return nil, err
			}
			template := &PipelineConfig{}
			err = json.Unmarshal(templateString, &template)
			if err != nil {
				return nil, err
			}
			spliter = CreatePipelineConfigSpliter(template)
			a.spliters[myId] = spliter
		}
	}
	return spliter, nil
}

func ActivityId(ctx activity.Context) string {
	return fmt.Sprintf("%s_%s", ctx.FlowDetails().Name(), ctx.TaskName())
}

type Deployment struct {
	ID              string
	DataFlow        []interface{}
	PipelineConfigs []*PipelineConfig
}

type Contribute struct {
	Ref      string `json:"ref"`
	Location string `json:"s3location"`
}

// Def is the configuration for the App
type PipelineConfig struct {
	app.Config
	//[{"ref":"git.tibco.com/git/product/ipaas/wi-contrib.git/contributions/General","s3location":"Tibco/General"},{"ref":"github.com/project-flogo/contrib/activity/log","s3location":"{USERID}/Default/activity/log"},{"ref":"github.com/TIBCOSoftware/GraphBuilder_Tools","s3location":"{USERID}/GraphBuilder_Tools"},{"ref":"github.com/TIBCOSoftware/ModelOps","s3location":"{USERID}/ModelOps"}]
	Contrib          map[string]Contribute
	Flows            map[string]*definition.DefinitionRep
	RunnerProperties map[string]interface{}
}

func (this *PipelineConfig) MarshalJSON() ([]byte, error) {

	fmt.Println("---  out Properties  --------")
	fmt.Println(this.Properties)
	fmt.Println("-------------------------")

	fmt.Println("---  out Contrib  --------")
	fmt.Println(this.Contrib)
	fmt.Println("-------------------------")

	ContribArray := make([]Contribute, 0)
	for _, value := range this.Contrib {
		ContribArray = append(ContribArray, value)
	}

	fmt.Println("---  out ContribArray  ---")
	fmt.Println(ContribArray)
	fmt.Println("--------------------------")

	this.MarshalResource()
	ContribArrayBytes, _ := json.Marshal(ContribArray)
	ContribArrayString := b64.URLEncoding.EncodeToString(ContribArrayBytes)

	fmt.Println("---  out flogoContrib  ---")
	fmt.Println(ContribArrayString)
	fmt.Println("--------------------------")

	return json.Marshal(&struct {
		app.Config
		ContribString string `json:"contrib"`
	}{
		Config:        this.Config,
		ContribString: ContribArrayString,
	})
}

func (this *PipelineConfig) UnmarshalJSON(data []byte) error {
	alias := &struct {
		app.Config
		ContribString string `json:"contrib"`
	}{}

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	this.Config = alias.Config

	fmt.Println("---  in Config  -------->>>>>>")
	fmt.Println(alias.Config)
	fmt.Println("------------------------>>>>>>")

	fmt.Println("---  in Properties  --------")
	fmt.Println(this.Properties)
	fmt.Println("-------------------------")

	this.UnmarshalResource()

	fmt.Println("---  in flogoContrib  ---")
	fmt.Println(alias.ContribString)
	fmt.Println("-------------------------")

	contributeArrayString, _ := b64.StdEncoding.DecodeString(alias.ContribString)
	contributeArray := make([]Contribute, 0)

	err := json.Unmarshal(contributeArrayString, &contributeArray)
	if err != nil {
		return err
	}

	fmt.Println("--- in contributeArray  ------")
	fmt.Println(contributeArray)
	fmt.Println("------------------------------")

	this.Contrib = make(map[string]Contribute)
	for _, contrib := range contributeArray {
		fmt.Println(contrib)
		this.Contrib[contrib.Ref] = contrib
		fmt.Println(this.Contrib)
	}

	fmt.Println("---  in Contrib  --------")
	fmt.Println(this.Contrib)
	fmt.Println("-------------------------")

	return nil
}

func (this *PipelineConfig) GetFlows() map[string]*definition.DefinitionRep {
	return this.Flows
}

func (this *PipelineConfig) MarshalResource() error {
	this.Resources = make([]*resource.Config, len(this.Flows))
	index := 0
	for id, aFlow := range this.Flows {
		flowDefBytes, err := json.Marshal(aFlow)
		if err != nil {
			return fmt.Errorf("error marshal resource : %s", err.Error())
		}
		this.Resources[index] = &resource.Config{
			ID:   id,
			Data: flowDefBytes,
		}
	}
	return nil
}

func (this *PipelineConfig) UnmarshalResource() error {
	this.Flows = make(map[string]*definition.DefinitionRep)
	for _, aResource := range this.Resources {
		flowDefBytes := aResource.Data
		var aFlow *definition.DefinitionRep
		err := json.Unmarshal(flowDefBytes, &aFlow)
		if err != nil {
			return fmt.Errorf("error loading flow resource with id '%s': %s", aResource.ID, err.Error())
		}
		this.Flows[aFlow.Name] = aFlow
		log.Info("?????????????????????")
		for _, task := range aFlow.Tasks {
			log.Info(task.ActivityCfgRep.Settings)
		}
		log.Info("?????????????????????")
	}
	return nil
}

func CreatePipelineConfigSpliter(Template *PipelineConfig) *PipelineConfigSpliter {
	return &PipelineConfigSpliter{
		Template:            Template,
		GRPCTrigger:         Template.Triggers[0],
		GRPCJsonDeserialize: Template.Flows["Pipeline Flow"].Tasks[0],
		GRPCJsonSerialize:   Template.Flows["Pipeline Flow"].Tasks[1],
		GRPCCoupler:         Template.Flows["Pipeline Flow"].Tasks[2],
		GRPCReturn:          Template.Flows["Pipeline Flow"].Tasks[3],
		Metadata:            Template.Flows["Pipeline Flow"].Metadata,
		Schemas:             Template.Schemas,
	}
}

type PipelineConfigSpliter struct {
	Template            *PipelineConfig
	GRPCTrigger         *trigger.Config
	GRPCJsonDeserialize *definition.TaskRep
	GRPCJsonSerialize   *definition.TaskRep
	GRPCCoupler         *definition.TaskRep
	GRPCReturn          *definition.TaskRep
	Metadata            *metadata.IOMetadata
	Schemas             map[string]*schema.Def
}

func (this *PipelineConfigSpliter) Split(aPipelineConfig *PipelineConfig) (*Deployment, error) {
	flogoImports := aPipelineConfig.Imports
	for _, importElement := range this.Template.Imports {
		flogoImports = append(flogoImports, importElement)
	}
	flogoNameBase := aPipelineConfig.Name
	flogoType := aPipelineConfig.Type
	flogoVersion := aPipelineConfig.Version
	flogoDescription := aPipelineConfig.Description
	flogoProperties := aPipelineConfig.Properties
	flogoContrib := aPipelineConfig.Contrib
	for contribKey, contrib := range this.Template.Contrib {
		flogoContrib[contribKey] = contrib
	}
	appModel := aPipelineConfig.AppModel
	var newPipelineConfig *PipelineConfig
	pipelineConfigs := make([]*PipelineConfig, 0)
	for _, aFlow := range aPipelineConfig.GetFlows() {
		for index, flow := range this.splitFlow(aFlow) {
			if 0 == index {
				flow.ExplicitReply = aFlow.ExplicitReply
				flow.Metadata = aFlow.Metadata
				newPipelineConfig = &PipelineConfig{
					Config: app.Config{
						Imports:     flogoImports,
						Name:        fmt.Sprintf("%s_%d", flogoNameBase, index),
						Type:        flogoType,
						Version:     flogoVersion,
						Description: flogoDescription,
						AppModel:    appModel,
						Triggers:    aPipelineConfig.Triggers,
						Schemas:     aPipelineConfig.Schemas,
						Properties:  flogoProperties,
					},
					Flows: map[string]*definition.DefinitionRep{
						aFlow.Name: flow,
					},
					Contrib: flogoContrib,
				}
			} else {
				if "Model Inference Flow" == flow.Name {
					log.Info("flow.Tasks[0] : ", flow.Tasks[0])
					newPipelineConfig = &PipelineConfig{
						Config: app.Config{
							Name: fmt.Sprintf("%s_%d", flogoNameBase, index),
							Type: ComponentType_External,
						},
						RunnerProperties: flow.Tasks[0].ActivityCfgRep.Settings,
					}
				} else {
					flow.ExplicitReply = true
					flow.Tasks = append(flow.Tasks, this.GRPCCoupler)
					flow.Tasks = append(flow.Tasks, this.GRPCReturn)
					flow.Metadata = this.Metadata
					newPipelineConfig = &PipelineConfig{
						Config: app.Config{
							Imports:     flogoImports,
							Name:        fmt.Sprintf("%s_%d", flogoNameBase, index),
							Type:        ComponentType_Flogo,
							Version:     flogoVersion,
							Description: flogoDescription,
							AppModel:    appModel,
							Triggers:    []*trigger.Config{this.GRPCTrigger},
							Schemas:     this.Schemas,
							Properties:  flogoProperties,
						},
						Flows: map[string]*definition.DefinitionRep{
							aFlow.Name: flow,
						},
						Contrib: flogoContrib,
					}
				}
			}
			pipelineConfigs = append(pipelineConfigs, newPipelineConfig)
		}

		fmt.Print("  - Links : ")
		for _, link := range aFlow.Links {
			fmt.Println(link)
		}

		fmt.Print("  - ErrorHandler : ")
		fmt.Println(aFlow.ErrorHandler)
	}

	var upstream string
	dataFlow := make([]interface{}, len(pipelineConfigs)-1)
	for index, pipelineConfig := range pipelineConfigs {
		if "" != upstream {
			dataFlow[index-1] = make(map[string]interface{})
			dataFlow[index-1].(map[string]interface{})["Upstream"] = upstream
			dataFlow[index-1].(map[string]interface{})["Downstream"] = pipelineConfig.Name
			upstream = pipelineConfig.Name
		} else {
			upstream = pipelineConfig.Name
		}
	}

	deployment := &Deployment{
		ID:              flogoNameBase,
		DataFlow:        dataFlow,
		PipelineConfigs: pipelineConfigs,
	}

	return deployment, nil
}

func (this *PipelineConfigSpliter) splitFlow(aFlow *definition.DefinitionRep) []*definition.DefinitionRep {
	flows := make([]*definition.DefinitionRep, 0)
	index := 0
	for _, task := range aFlow.Tasks {
		log.Info("Task.ID : ", task.ID)
		if "#modelrunner" == task.ActivityCfgRep.Ref {
			for _, replacement := range this.modelRunnerReplacement(task) {
				flows[index].Tasks = append(flows[index].Tasks, replacement)
			}
			index += 1
			log.Info("Task : ", task)
			flows = append(flows, &definition.DefinitionRep{
				Name:    "Model Inference Flow",
				ModelID: aFlow.ModelID,
				Tasks:   []*definition.TaskRep{task}},
			)
			index += 1
		} else if "#actreturn" == task.ActivityCfgRep.Ref {
			task.ActivityCfgRep.Settings["mappings"].(map[string]interface{})["Content"] = "=$activity[MLpipelinecoupler].Reply.Content"
			flows[0].Tasks = append(flows[0].Tasks, task)
		} else {
			if index == len(flows) {
				flows = append(flows, &definition.DefinitionRep{
					Name:    aFlow.Name,
					ModelID: aFlow.ModelID,
					Tasks:   make([]*definition.TaskRep, 0)},
				)
			}
			flows[index].Tasks = append(flows[index].Tasks, task)
		}
	}
	return flows
}

func (this *PipelineConfigSpliter) modelRunnerReplacement(modelRunner *definition.TaskRep) []*definition.TaskRep {
	tasks := make([]*definition.TaskRep, 2)

	jsonSerialize := &definition.TaskRep{}
	deepCopy(*this.GRPCJsonSerialize, jsonSerialize)
	jsonSerialize.ActivityCfgRep.Input["Data"] = modelRunner.ActivityCfgRep.Input["DataIn"]
	jsonSerialize.ActivityCfgRep.Schemas.Input["Data"] = modelRunner.ActivityCfgRep.Schemas.Input["DataIn"]
	tasks[0] = jsonSerialize

	grpcCoupler := &definition.TaskRep{}
	deepCopy(*this.GRPCCoupler, grpcCoupler)
	tasks[1] = grpcCoupler

	return tasks
}

func insertTask(slice []*definition.TaskRep, index int, element *definition.TaskRep) []*definition.TaskRep {
	slice = append(slice, &definition.TaskRep{})
	copy(slice[index+1:], slice[index:])
	slice[index] = element
	return slice
}

func clone(a, b interface{}) {
	buff := new(bytes.Buffer)
	enc := gob.NewEncoder(buff)
	dec := gob.NewDecoder(buff)
	enc.Encode(a)
	dec.Decode(b)
}

func deepCopy(a, b interface{}) {
	byt, _ := json.Marshal(a)
	json.Unmarshal(byt, b)
}
