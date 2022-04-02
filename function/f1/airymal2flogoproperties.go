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
		log.Warn("(fnAirYmal2FlogoProperties.Eval) Illegal parameter : Unable to parse yaml.")
		return nil, nil
	}

	return objectbuilder.Ymal2FlogoProperties(yamlDescriptor), nil
}
