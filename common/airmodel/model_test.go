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

	descriptorString, _, _, _, err := BuildFlogoApp(
		templateLibrary,
		"test_pipeline",
		applicationPipelineDescriptor,
		map[string]interface{}{},
		[]interface{}{},
	)
	//	fmt.Println("descriptorString ====", descriptorString)
	fmt.Println("err ====", err)

	_ = ioutil.WriteFile(fmt.Sprintf("./%s_flogo.json", appname), []byte(descriptorString), 0644)
}
