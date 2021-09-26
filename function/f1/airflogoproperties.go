package f1

import (
	"encoding/json"
	"fmt"

	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/data/expression/function"
)

func init() {
	function.Register(&fnAirFlogoProperties{})
}

type fnAirFlogoProperties struct {
}

func (fnAirFlogoProperties) Name() string {
	return "airflogoproperties"
}

func (fnAirFlogoProperties) Sig() (paramTypes []data.Type, isVariadic bool) {
	return []data.Type{data.TypeString}, false
}

/*
{
    "properties": [
        {
            "name": "Logging_LogLevel",
            "type": "string",
            "value": "INFO"
        }
    ]
}
*/

func (fnAirFlogoProperties) Eval(params ...interface{}) (interface{}, error) {
	flogoApp, ok1 := params[0].(string)
	if !ok1 {
		return nil, fmt.Errorf("Illegal parameter : flogoApp json string")
	}

	var flogoAppDescriptor map[string]interface{}
	err := json.Unmarshal([]byte(flogoApp), &flogoAppDescriptor)
	if nil != err {
		return true, err
	}

	properties := make([]map[string]interface{}, 0)
	for _, property := range flogoAppDescriptor["properties"].([]interface{}) {
		propertyObj := property.(map[string]interface{})
		properties = append(properties, map[string]interface{}{
			"Name":  propertyObj["name"],
			"Value": propertyObj["value"],
			"Type":  propertyObj["type"],
		})
	}

	return properties, nil
}
