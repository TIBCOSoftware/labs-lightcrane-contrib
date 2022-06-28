package airpipelinebuilder2

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/support/test"
	"github.com/stretchr/testify/assert"
)

var activityMetadata *activity.Metadata

//const connFile = "mqtt_conn_cacert.json"

const connFile = "mqtt_conn.json"

func getActivityMetadata() *activity.Metadata {

	if activityMetadata == nil {
		jsonMetadataBytes, err := ioutil.ReadFile("activity.json")
		if err != nil {
			panic("No Json Metadata found for activity.json path")
		}

		activityMetadata = activity.NewMetadata(string(jsonMetadataBytes))
	}

	return activityMetadata
}

func TestCreate(t *testing.T) {

	act := NewActivity(getActivityMetadata())

	if act == nil {
		t.Error("Activity Not Created")
		t.Fail()
		return
	}
}

func TestEval(t *testing.T) {
	//log.SetLogLevel(logger.DebugLevel)
	//log.SetLogLevel(logger.InfoLevel)

	act := NewActivity(getActivityMetadata())
	tc := test.NewTestActivityContext(getActivityMetadata())

	//setup attrs
	tc.SetSetting("TemplateFolder", "../../../../services/labs-lightcrane-services/air/airpipeline_oss")

	fileContent, err := ioutil.ReadFile("request01.json")
	if err != nil {
		t.Errorf("Could not get air descriptor: %s", err)
		t.Fail()
	}
	var descriptor map[string]interface{}
	json.Unmarshal(fileContent, &descriptor)
	airDescriptor := descriptor["AirDescriptor"]
	airDescriptor.(map[string]interface{})["properties"] = []interface{}{}
	tc.SetInput("AirDescriptor", airDescriptor)

	done, err := act.Eval(tc)

	assert.Nil(t, err)
	if !done || err != nil {
		t.Errorf("Activity failed : %s", err)
		t.Fail()
	}
}
