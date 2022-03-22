package f1

import (
	"fmt"
	"strings"

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
		log.Info("(fnAirYmal2FlogoProperties.Eval) Illegal parameter : Unable to parse yaml.")
		return nil, nil
	}

	handler := AirYmal2FlogoProperties{properties: make(map[string]interface{})}
	walker := objectbuilder.NewGOLangObjectWalker(handler)
	walker.Start(yamlDescriptor)
	propArray := make([]interface{}, 0)
	for _, prop := range handler.GetData() {
		propArray = append(propArray, prop)
	}
	return propArray, nil
}

type AirYmal2FlogoProperties struct {
	properties map[string]interface{}
}

func (a AirYmal2FlogoProperties) HandleElements(namespace objectbuilder.ElementId, element interface{}, dataType interface{}) interface{} {
	log.Info("name space : ", namespace.GetId(), ", element = ", element, ", dataType = ", dataType)
	if "[]interface{}" != dataType && "map[string]interface{}" != dataType {
		name := namespace.GetId()[0]
		log.Info("(fnAirYmal2FlogoProperties.HandleElements) a.properties : Name = ", name[strings.Index(name, ".")+1:], ", Value = ", element)
		if "string" == dataType {
			element = fmt.Sprintf("'%s'", element)
		}
		a.properties[name[strings.Index(name, ".")+1:]] = element
		log.Info("(fnAirYmal2FlogoProperties.HandleElements) a.properties = ", a.properties)
	}
	return nil
}

func (a AirYmal2FlogoProperties) GetData() []map[string]interface{} {
	log.Info("(fnAirYmal2FlogoProperties.GetData) a.properties = ", a.properties)
	propertiesArray := make([]map[string]interface{}, 0)
	for name, value := range a.properties {
		propertiesArray = append(propertiesArray, map[string]interface{}{
			"Name":  name,
			"Value": value,
		})
	}
	log.Info("(fnAirYmal2FlogoProperties.GetData) propertiesArray = ", propertiesArray)
	return propertiesArray
}
