package f1

import (
	"github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/util"
	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/data/expression/function"
)

func init() {
	function.Register(&fnCombineProperties{})
}

type fnCombineProperties struct {
}

func (fnCombineProperties) Name() string {
	return "combineproperties"
}

func (fnCombineProperties) Sig() (paramTypes []data.Type, isVariadic bool) {
	return []data.Type{data.TypeArray}, true
}

func (fnCombineProperties) Eval(params ...interface{}) (interface{}, error) {
	valueMap := make(map[string]interface{})
	typeMap := make(map[string]interface{})
	for _, param := range params {
		if nil != param {
			for _, prop := range param.([]interface{}) {
				name := util.GetPropertyElementAsString("Name", prop)
				value := util.GetPropertyElement("Value", prop)
				dtype := util.GetPropertyElement("Type", prop)
				if "" != name {
					valueMap[name] = value
					typeMap[name] = dtype
				}
			}
		}
	}

	combined := make([]interface{}, 0)
	for name, value := range valueMap {
		combined = append(combined, map[string]interface{}{
			"Name":  name,
			"Value": value,
			"Type":  typeMap[name],
		})
	}

	return combined, nil
}
