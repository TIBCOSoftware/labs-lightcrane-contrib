package airparameterbuilder

import (
	//"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/TIBCOSoftware/flogo-contrib/action/flow/test"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
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
	log.SetLogLevel(logger.InfoLevel) //.DebugLevel)

	descriptor := "{\"source\": {\"name\": \"MQTT\",\"properties\" : [{\"Name\":\"key1\",\"Value\":\"value1\"}]},\"logic\": [{\"name\": \"Dgraph\",\"properties\" : [{\"Name\":\"key1\",\"Value\":\"value1\"}]}]}"
	//descriptor := "{\"source\": {\"name\": \"Kafka\",\"properties\" : [{\"Name\":\"key1\",\"Value\":\"value1\"}]},\"logic\": [{\"name\": \"Postgres\",\"properties\" : [{\"Name\":\"key1\",\"Value\":\"value1\"}]}]}"

	act := NewActivity(getActivityMetadata())
	tc := test.NewTestActivityContext(getActivityMetadata())

	//setup attrs
	tc.SetSetting("TemplateFolder", "../../../../../../../../services/builder/docker/airpipeline/")
	tc.SetInput("AirDescriptor", descriptor)

	_, err := act.Eval(tc)
	assert.Nil(t, err)
	if err != nil {
		t.Errorf("Could not publish a message: %s", err)
		t.Fail()
	}
}
