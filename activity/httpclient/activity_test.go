package httpclient

import (
	"encoding/base64"
	"encoding/json"
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

func TestInference(t *testing.T) {
	log.SetLogLevel(logger.DebugLevel)

	act := NewActivity(getActivityMetadata())
	tc := test.NewTestActivityContext(getActivityMetadata())

	var data interface{}
	_ = json.Unmarshal([]byte("{\"device\":\"RESTDevice\",\"gateway\":\"HelloWorldGroup\",\"id\":\"fa9f3c8d-e4ec-4d48-b6f0-def16a5f1523\",\"readings\":[{\"deviceName\":\"RESTDevice\",\"id\":\"8b5080ba-f3f8-427a-8429-ac5cf6d2e823\",\"mediaType\":\"image/jpeg\",\"origin\":1650595536601,\"profileName\":\"Generic-REST-Device\",\"resourceName\":\"image_reading\",\"value\":\"/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAP//////////////////////////////////////////////////////////////////////////////////////2wBDAf//////////////////////////////////////////////////////////////////////////////////////wAARCADqATkDASIAAhEBAxEB/8QAFwABAQEBAAAAAAAAAAAAAAAAAAECA//EACQQAQEBAAIBBAMBAQEBAAAAAAABESExQQISUXFhgZGxocHw/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAH/xAAWEQEBAQAAAAAAAAAAAAAAAAAAEQH/2gAMAwEAAhEDEQA/AMriLyCKgg1gQwCgs4FTMOdutepjQak+FzMSVqgxZdRdPPIIvH5WzzGdBriphtTeAXg2ZjKA1pqKDUGZca3foBek8gFv8Ie3fKdA1qb8s7hoL6eLVt51FsAnql3Ut1M7AWbflLMDkEMX/F6/YjK/pADFQAUNA6alYagKk72m/j9p4Bq2fDDSYKLNXPNLoHE/NT6RYC31cJxZ3yWVM+aBYi/S2ZgiAsnYJx5D21vPmqrm3PTfpQQwyAC8JZvSKDni41ZrMuUVVl+Uz9w9v/1QWrZsZ5nFPHYH+JZyureQSF5M+fJ0CAfwRAVRBQA1DAWVUayoJUWoDpsxntPsueBV4+VxhdyAtv8AjOLGpIDMLbeGvbF4iozJfr/WukAVABAXAQXEAAASzVAZdO2WNordm+emFl7XcQSNZiFtv0C9w90nhJf4mA1u+GcJFwIyAqL/AOovwgGNfSRqdIrNa29M0gKCAojU9PAMjWXpckEJFNFEAAXEUBABYz6rZ0ureQc9vyt9XxDF2QAXtABcQAs0AZywkvluJbyipifas52DcyxjlZweAO0xri/hc+wZOEKIu6nSyeToVZyWXwvCg53gW81QQ7aTNAn5dGZJPs1UXURQAUEMCXQLZE93PRZ5hPTgNMrbIzKCm52LZwCs+2M8w2g3sjPuZAXb4IsMAUACzVUGM4/K+md6vEXUUyM5PDR0IxYe6ramih0VNBrS4xoqN8Q1BFQk3yqyAsioioAAKgDSJL4/jQIn5igLrPqtOuf6oOaxbMoAltUAhhIoJiiggrPu+AaOIxtAX3JbaAIaLwi4t9X4T3fg2AFtqcrUUarP20zUDAmqoE0WRBZPNVUVEAAAAVAC8kvih2DSKxOdBqs7Z0l0gI0mKAC4AuHE7ZtBriM+744QAAAAABAFsveIttBICyaikvy1+r/Cen5rWQHIBQa4rIDRqSl5qDWqziqgAAAATA7BpGdqXb2C2+J/UgAtRQBSQtkBWb6vhLbQAAAAAEBRAAAAAUbm+GZNdPxAP+ql2Tjwx7/wIgZ8iKvBk+CJoCXii9gaqZ/qqihAAAEVABGkBFUwBftNkZ3QW34QAAABFAQAVAAAAAARVkl8gs/43sk1jL45LvHArepk+E9XTG35oLqsmIKmLAEygKg0y1AFQBUXwgAAAoBC34S3UAAABAVAAAAAABAUQAVABdRQa1PcYyit2z58M8C4ouM2NXpOEGeWtNZUatiAIoAKIoCoAoG4C9MW6dgIoAIAAAAAAACKWAgL0CAAAALiANCKioNLgM1CrLihmTafkt1EF3SZ5ZVUW4mnIKvAi5fhEURVDWVQBRAAAAAAAAQFRVyAyulgAqCKlF8IqLsEgC9mGoC+IusqCrv5ZEUVOk1RuJfwSLOOkGFi4XPCoYYrNiKauosBGi9ICstM1UAAAAAAFQ0VcTBAXUGgIqGoKhKAzRRUQUAwxoSrGRpkQA/qiosOL9oJptMRRVZa0VUqSiChE6BqMgCwqKqIogAIAqKCKgKoogg0lBFuIKgAAAKNRlf2gqsftsEtZWoAAqAACKoMqAAeSoqp39kL2AqLOlE8rEBFQARYALhigrNC9gGmooLp4TweEQFFBFAECgIoAu0ifIAqAAA//9k=\",\"valueType\":\"String\"}],\"source\":1650595536601743000}"), &data)
	value, _ := base64.StdEncoding.DecodeString(data.(map[string]interface{})["readings"].([]interface{})[0].(map[string]interface{})["value"].(string))

	//setup attrs
	tc.SetSetting("method", "POST")
	tc.SetSetting("timeout", 100000)
	var urlMapping interface{}
	_ = json.Unmarshal([]byte("[{\"Alias\":\"0\",\"URL\":\"https://tibcocvmodel-prediction.cognitiveservices.azure.com/customvision/v3.0/Prediction/c0f2ffe4-bb8f-4f09-a552-b85332af6769/classify/iterations/Iteration1/image\"}]"), &urlMapping)
	tc.SetSetting("urlMapping", urlMapping)
	tc.SetSetting("leftToken", "$")
	tc.SetSetting("rightToken", "$")
	//	tc.SetSetting("variablesDef", "")
	var httpHeaders interface{}
	_ = json.Unmarshal([]byte("[{\"Key\":\"Accept\",\"Value\":\"application/json\"},{\"Key\":\"Content-Type\",\"Value\":\"application/json-patch+json\"}]"), &httpHeaders)
	tc.SetSetting("httpHeaders", httpHeaders)

	tc.SetInput("URL", "0")
	var headers interface{}
	_ = json.Unmarshal([]byte("[{\"key\":\"Accept\",\"value\":\"*/*\"},{\"key\":\"Accept-Encoding\",\"value\":\"gz, defalte, br\"},{\"key\":\"Connection\",\"value\":\"keep-alive\"},{\"key\":\"Prediction-Key\",\"value\":\"3589909f8bd04276acda68e253dfccba\"},{\"key\":\"Content-Type\",\"value\":\"application/octet-stream\"}]"), &headers)
	tc.SetInput("Headers", headers)
	//	tc.SetInput("Method", properties)
	tc.SetInput("Body", value)
	//	tc.SetInput("Variables", properties)
	tc.SetInput("SkipCondition", false)

	_, err := act.Eval(tc)
	assert.Nil(t, err)
	if err != nil {
		t.Logf("Could not publish a message: %s", err)
		//		t.Fail()
	}
}
