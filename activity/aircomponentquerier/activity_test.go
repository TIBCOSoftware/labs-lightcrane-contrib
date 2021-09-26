package aircomponentquerier

import (
	"io/ioutil"
	"testing"

	"github.com/TIBCOSoftware/flogo-contrib/action/flow/test"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
)

var activityMetadata *activity.Metadata

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
	log.SetLogLevel(logger.InfoLevel)

	act := NewActivity(getActivityMetadata())
	tc := test.NewTestActivityContext(getActivityMetadata())

	//setup attrs
	tc.SetSetting("TemplateFolder", "../../../../../../../../services/air/docker/airpipeline")
	tc.SetInput("Category", "*")
	tc.SetInput("Component", nil)

	result, err := act.Eval(tc)
	log.Info("result = ", result)
	if err != nil {
		t.Errorf("Could not publish a message: %s", err)
		t.Fail()
	}
}
