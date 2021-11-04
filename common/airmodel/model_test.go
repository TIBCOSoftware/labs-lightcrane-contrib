package model

import (
	"io/ioutil"
	"strings"
	"testing"

	"fmt"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

const (
	iPorts      = "ports"
	iProperties = "properties"
)

func TestBuildTemplates(t *testing.T) {

	templateLibrary, err := NewFlogoTemplateLibrary("../../../../../../services/air/docker/airpipeline")
	if nil != err {
		t.Fatalf("Error : %v", err)
	}

	applicationPipelineDescriptor, _ := FromFile("./air_description_expression.json")

	appPropertiesByComponent := make([]interface{}, 0)
	var appProperties []interface{}
	pipeline := templateLibrary.GetPipeline()

	sourceObj := applicationPipelineDescriptor["source"].(map[string]interface{})
	longname := sourceObj["name"].(string)
	category := longname[:strings.Index(longname, ".")]
	name := longname[strings.Index(longname, ".")+1:]
	dataSource := templateLibrary.GetComponent(-1, category, name).(DataSource)
	pipeline.SetDataSource(dataSource)

	appProperties = make([]interface{}, 0)
	if nil != sourceObj[iProperties] {
		for _, property := range sourceObj[iProperties].([]interface{}) {
			appProperties = append(appProperties, map[string]interface{}{
				"Name":  util.GetPropertyElement("Name", property),
				"Value": util.GetPropertyElement("Value", property),
				"Type":  util.GetPropertyElement("Type", property),
			})
		}
	}
	appPropertiesByComponent = append(appPropertiesByComponent, appProperties)

	for key, value := range applicationPipelineDescriptor {
		switch key {
		case "logic":
			logicArray := value.([]interface{})
			for index, logic := range logicArray {
				logicObj := logic.(map[string]interface{})
				longname := logicObj["name"].(string)
				category := longname[:strings.Index(longname, ".")]
				name := longname[strings.Index(longname, ".")+1:]
				logic := templateLibrary.GetComponent(index, category, name).(Logic)
				pipeline.AddLogic(logic)

				appProperties = make([]interface{}, 0)
				if nil != logicObj[iProperties] {
					for _, property := range logicObj[iProperties].([]interface{}) {
						appProperties = append(appProperties, map[string]interface{}{
							"Name":  util.GetPropertyElement("Name", property),
							"Value": util.GetPropertyElement("Value", property),
							"Type":  util.GetPropertyElement("Type", property),
						})
					}
				}
				appPropertiesByComponent = append(appPropertiesByComponent, appProperties)
			}
		}
	}

	descriptorString, _ := pipeline.Build()
	fmt.Println("====", descriptorString)
	propertyContainer := pipeline.GetProperties()
	appProperties = propertyContainer.GetReplacements(appPropertiesByComponent)
	fmt.Println("====", appProperties)

	_ = ioutil.WriteFile("./AirPipeline.json", []byte(descriptorString), 0644)
}
