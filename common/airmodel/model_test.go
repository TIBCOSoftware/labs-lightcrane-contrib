package airmodel

import (
	"io/ioutil"
	"testing"

	"fmt"
)

func TestBuildTemplates(t *testing.T) {

	templateLibrary, err := NewFlogoTemplateLibrary("../../../../services/labs-lightcrane-services/air/airpipeline_oss")
	if nil != err {
		t.Fatalf("Error : %v", err)
	}

	appname := "air_description_sample01"

	applicationPipelineDescriptor, _ := FromFile(fmt.Sprintf("./%s.json", appname))
	applicationPipelineDescriptor["properties"] = []interface{}{
		map[string]interface{}{
			"Name":  "FLOGO_APP_PROPS_ENV",
			"Value": "auto",
		},
	}
	fmt.Println("applicationPipelineDescriptor before ====", applicationPipelineDescriptor)

	descriptorString, _, _, _, _, err := BuildFlogoApp(
		templateLibrary,
		"test_pipeline",
		applicationPipelineDescriptor,
		[]interface{}{},
		map[string]interface{}{
			"HA": map[string]interface{}{
				"controllerType":       "InMemory",
				"controllerProperties": map[string]interface{}{},
				"replicas":             3,
			},
		},
	)
	fmt.Println("applicationPipelineDescriptor after ====", applicationPipelineDescriptor)
	fmt.Println("err ====", err)

	_ = ioutil.WriteFile(fmt.Sprintf("./%s_flogo.json", appname), []byte(descriptorString), 0644)
}
