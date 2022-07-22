package airmodel

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"fmt"
)

func TestBuildTemplates(t *testing.T) {

	templateLibrary, err := NewFlogoTemplateLibrary("../../../../services/labs-lightcrane-services/air/airpipeline_oss")
	if nil != err {
		t.Fatalf("Error : %v", err)
	}

	appname := "air_description_sample03"

	applicationPipelineDescriptor, _ := FromFile(fmt.Sprintf("./%s.json", appname))
	applicationPipelineDescriptor["properties"] = []interface{}{
		map[string]interface{}{
			"Name":  "FLOGO_APP_PROPS_ENV",
			"Value": "auto",
		},
	}
	fmt.Println("applicationPipelineDescriptor before ====", applicationPipelineDescriptor)

	descriptorString, pipeline, extra, _, _, replicas, err := BuildFlogoApp(
		templateLibrary,
		"test_pipeline",
		applicationPipelineDescriptor,
		map[string]interface{}{
			"HA": map[string]interface{}{
				"controllerProperties": map[string]interface{}{
					"tableType": "InMemory",
				},
				"replicas": float64(3),
			},
		},
	)
	fmt.Println("applicationPipelineDescriptor after ====", applicationPipelineDescriptor)
	fmt.Println("extra ====", extra)
	fmt.Println("replicas ====", replicas)
	fmt.Println("err ====", err)
	controllerPropertiesByte, err := json.Marshal(applicationPipelineDescriptor)
	_ = ioutil.WriteFile("./applicationPipelineDescriptor.json", []byte(controllerPropertiesByte), 0644)
	pipelineByte, err := json.Marshal(pipeline)
	_ = ioutil.WriteFile("./pipeline.json", []byte(pipelineByte), 0644)
	_ = ioutil.WriteFile(fmt.Sprintf("./%s_flogo.json", appname), []byte(descriptorString), 0644)
}
