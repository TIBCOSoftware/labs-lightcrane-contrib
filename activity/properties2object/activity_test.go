package properties2object

import (
	"encoding/json"
	"fmt"
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

	//	properties := "[{\"Name\":\"version\",\"Type\":\"\",\"Value\":\"3.6\"},{\"Name\":\"services.singlecellviability0.container_name\",\"Type\":\"\",\"Value\":\"Air-LC_SingleCellViability0_singlecellviability0\"},{\"Name\":\"services.singlecellviability0.build\",\"Type\":\"\",\"Value\":\"001\"},{\"Name\":\"services.singlecellviability0.deploy.resources.reservations.devices[0].driver\",\"Type\":\"\",\"Value\":\"nvidia\"},{\"Name\":\"services.singlecellviability0.deploy.resources.reservations.devices[0].count\",\"Type\":\"\",\"Value\":\"1\"},{\"Name\":\"services.singlecellviability0.deploy.resources.reservations.devices[0].capabilities[0]\",\"Type\":\"\",\"Value\":\"gpu\"},{\"Name\":\"services.singlecellviability0.volumes[0]\",\"Type\":\"String\",\"Value\":\"/home/syang/Data:/data\"},{\"Name\":\"services.singlecellviability0.environment[0]\",\"Type\":null,\"Value\":\"System_ID=$ID$\"},{\"Name\":\"services.singlecellviability0.environment[1]\",\"Type\":null,\"Value\":\"System_ServiceLocator=$ServiceLocator$\"},{\"Name\":\"services.singlecellviability0.environment[2]\",\"Type\":null,\"Value\":\"System_ExternalEndpointIP=192.168.1.200\"},{\"Name\":\"services.singlecellviability0.environment[3]\",\"Type\":null,\"Value\":\"System_ExternalEndpointPort=10103\"},{\"Name\":\"services.singlecellviability0.environment[4]\",\"Type\":null,\"Value\":\"System_EndpointComponent=Air-LC_SingleCellViability0\"},{\"Name\":\"services.singlecellviability0.environment[5]\",\"Type\":null,\"Value\":\"System_Standalone=True\"},{\"Name\":\"services.singlecellviability0.environment[6]\",\"Type\":null,\"Value\":\"System_EchoOn=True\"},{\"Name\":\"services.singlecellviability0.environment[7]\",\"Type\":null,\"Value\":\"Working_Folder=/app/artifacts\"},{\"Name\":\"services.singlecellviability0.environment[8]\",\"Type\":null,\"Value\":\"PythonModel_plugin=artifacts.inference\"},{\"Name\":\"services.singlecellviability0.ports[0]\",\"Type\":\"String\",\"Value\":\"10103:10100\"}]"
	propertiesStr := "[{\"Group\":\"main\",\"Value\":[{\"Name\":\"services.openvino.command[9]\",\"Type\":\"\",\"Value\":\"INFO\"},{\"Name\":\"version\",\"Type\":\"\",\"Value\":\"3.7\"},{\"Name\":\"services.openvino.command[7]\",\"Type\":\"\",\"Value\":\"9001\"},{\"Name\":\"services.openvino.container_name\",\"Type\":\"\",\"Value\":\"openvino\"},{\"Name\":\"services.openvino.ports[1]\",\"Type\":\"\",\"Value\":\"9901:9001\"},{\"Name\":\"services.openvino.command[1]\",\"Type\":\"\",\"Value\":\"/models/cyto2torch_0\"},{\"Name\":\"services.openvino.restart\",\"Type\":\"\",\"Value\":\"always\"},{\"Name\":\"services.openvino.command[8]\",\"Type\":\"\",\"Value\":\"--log_level\"},{\"Name\":\"services.openvino.command[4]\",\"Type\":\"\",\"Value\":\"--port\"},{\"Name\":\"services.openvino.command[3]\",\"Type\":\"\",\"Value\":\"cyto2torch_0\"},{\"Name\":\"services.openvino.volumes[0]\",\"Type\":\"String\",\"Value\":\"/Users/jumbo/Works/LightCrane/notebook/ValitaCell/ModelService/openvino/artifacts/model:/models/cyto2torch_0\"},{\"Name\":\"services.openvino.command[2]\",\"Type\":\"\",\"Value\":\"--model_name\"},{\"Name\":\"services.openvino.ports[0]\",\"Type\":\"\",\"Value\":\"9900:9000\"},{\"Name\":\"services.openvino.command[0]\",\"Type\":\"\",\"Value\":\"--model_path\"},{\"Name\":\"services.openvino.command[6]\",\"Type\":\"\",\"Value\":\"--rest_port\"},{\"Name\":\"services.openvino.user\",\"Type\":\"\",\"Value\":\"${UID}:${GID}\"},{\"Name\":\"services.openvino.command[5]\",\"Type\":\"\",\"Value\":\"9000\"},{\"Name\":\"services.openvino.image\",\"Type\":\"\",\"Value\":\"openvino/model_server:latest\"},{\"Name\":\"services.openvino.environment[0]\",\"Type\":null,\"Value\":\"App.LogLevel=INFO\"}]}]"
	var properties []interface{}
	_ = json.Unmarshal([]byte(propertiesStr), &properties)
	act := NewActivity(getActivityMetadata())
	tc := test.NewTestActivityContext(getActivityMetadata())

	//setup attrs
	tc.SetSetting("TemplateFolder", "../../../../../../../../services/builder/docker/airpipeline/")
	fmt.Println(properties)
	tc.SetInput("Properties", properties)

	_, err := act.Eval(tc)
	assert.Nil(t, err)
	if err != nil {
		t.Errorf("Could not publish a message: %s", err)
		t.Fail()
	}
}
