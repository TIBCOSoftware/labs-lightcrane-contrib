package f1

import (
	"fmt"
	"strings"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/data/expression/function"
)

func init() {
	function.Register(&fnModelArtifactMap{})
}

type fnModelArtifactMap struct {
}

func (fnModelArtifactMap) Name() string {
	return "modelartifactmap"
}

func (fnModelArtifactMap) Sig() (paramTypes []data.Type, isVariadic bool) {
	return []data.Type{data.TypeArray}, true
}

func (fnModelArtifactMap) Eval(params ...interface{}) (interface{}, error) {
	folderMap := make([]interface{}, 0)
	var index int = 0
	for _, prop := range params[0].([]interface{}) {
		name := util.GetPropertyElementAsString("Name", prop)
		value := util.GetPropertyElement("Value", prop)
		dtype := util.GetPropertyElement("Type", prop)
		if "folder" == dtype && false == strings.HasSuffix(strings.ToLower(name), ".zip") {
			folderMap = append(folderMap, map[string]interface{}{
				"Name":  fmt.Sprintf("services.$Name$.volumes[%d]", index),
				"Value": fmt.Sprintf("%s:%s", value, name),
				"Type":  "String",
			})
			index = index + 1
		}
	}

	return folderMap, nil
}
