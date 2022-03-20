package f1

import (
	"fmt"

	yaml "gopkg.in/yaml.v3"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/objectbuilder"
	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/data/expression/function"
)

func init() {
	function.Register(&fnAirYmal2FlogoProperties{})
}

type fnAirYmal2FlogoProperties struct {
}

func (f fnAirYmal2FlogoProperties) Name() string {
	return "airymal2flogoproperties"
}

func (f fnAirYmal2FlogoProperties) Sig() (paramTypes []data.Type, isVariadic bool) {
	return []data.Type{data.TypeString}, false
}

func (f fnAirYmal2FlogoProperties) Eval(params ...interface{}) (interface{}, error) {
	ymalString, ok1 := params[0].(string)
	if !ok1 {
		return nil, fmt.Errorf("Illegal parameter : require yaml string")
	}

	var yamlDescriptor map[string]interface{}
	err := yaml.Unmarshal([]byte(ymalString), &yamlDescriptor)
	if nil != err {
		return true, err
	}

	properties := make([]map[string]interface{}, 0)
	walker := objectbuilder.NewGOLangObjectWalker(f)
	walker.Start(yamlDescriptor)
	//	for _, property := range flogoAppDescriptor["properties"].([]interface{}) {
	//		propertyObj := property.(map[string]interface{})
	//		properties = append(properties, map[string]interface{}{
	//			"Name":  propertyObj["name"],
	//			"Value": propertyObj["value"],
	//			"Type":  propertyObj["type"],
	//		})
	//	}

	return properties, nil
}

func (f fnAirYmal2FlogoProperties) HandleElements(namespace objectbuilder.ElementId, element interface{}, dataType interface{}) interface{} {
	log.Info("name space : ", namespace.GetId())
	return nil
}

func (f fnAirYmal2FlogoProperties) GetData() []map[string]interface{} {
	return nil
}
