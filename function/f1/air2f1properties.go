package f1

import (
	"fmt"
	"strings"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/data/expression/function"
)

func init() {
	function.Register(&fnAir2F1Properties{})
}

type fnAir2F1Properties struct {
}

func (fnAir2F1Properties) Name() string {
	return "air2f1properties"
}

func (fnAir2F1Properties) Sig() (paramTypes []data.Type, isVariadic bool) {
	return []data.Type{data.TypeString, data.TypeString, data.TypeArray, data.TypeArray, data.TypeAny, data.TypeArray}, false
}

func (fnAir2F1Properties) Eval(params ...interface{}) (interface{}, error) {
	log.Debug("\n\n\n (fnAir2F1Properties.Eval) in deployType ========>", params[0])
	log.Debug("(fnAir2F1Properties.Eval) in prefix ========>", params[1])
	log.Debug("(fnAir2F1Properties.Eval) in f1Properties ========>", params[2])
	log.Debug("(fnAir2F1Properties.Eval) in airPropertiesOrig ========>", params[3])
	log.Debug("(fnAir2F1Properties.Eval) in propertyNameDef ========>", params[4])
	log.Debug("(fnAir2F1Properties.Eval) in extraProperties ========>", params[5])

	deployType := params[0].(string)
	prefix := params[1].(string)
	f1Properties, ok := params[2].([]interface{})
	if !ok {
		f1Properties = make([]interface{}, 0)
		log.Warn("Incoming f1Properties is null !!!")
	}
	airPropertiesOrig := params[3].([]interface{})
	propertyNameDef := params[4].(map[string]interface{})
	extraProperties := params[5].([]interface{})

	/*
		Property Name lookup and conversion :
		1. from w/o to w category
		2. conver all '.' to '_'
	*/
	for _, airProerty := range airPropertiesOrig {
		name := airProerty.(map[string]interface{})["Name"].(string)
		component := airProerty.(map[string]interface{})["Component"].(string)
		if nil != propertyNameDef[component] {
			log.Debug("(fnAir2F1Properties.Eval) in new name ========>", propertyNameDef[component].(map[string]interface{})[name])
			if nil != propertyNameDef[component].(map[string]interface{})[name] {
				airProerty.(map[string]interface{})["Name"] = strings.ReplaceAll(propertyNameDef[component].(map[string]interface{})[name].(string), ".", "_")
			}
		}
	}

	airProperties := make([]interface{}, 0)
	exist := make(map[string]bool)
	/*
		duplication fillter
	*/
	for _, property := range airPropertiesOrig {
		name := property.(map[string]interface{})["Name"].(string)
		if !exist[name] {
			airProperties = append(airProperties, property)
			exist[name] = true
		}
	}

	log.Debug("(fnAir2F1Properties.Eval) out airProperties ========>", airProperties)

	switch deployType {
	case "k8s":
		f1Properties = createK8sF1Properties(
			prefix,
			f1Properties,
			airProperties,
			extraProperties,
		)
	default:
		f1Properties = createDockerF1Properties(
			prefix,
			f1Properties,
			airProperties,
			extraProperties,
		)
	}

	log.Debug("(fnAir2F1Properties.Eval) out f1Properties ========>", f1Properties)
	return f1Properties, nil
}

func createK8sF1Properties(
	prefix string,
	f1PropertiesMaster []interface{},
	airProperties []interface{},
	extraProperties []interface{},
) []interface{} {
	/*
		Only main properties allows to be modified
		[map[
			Group:main
			Value:[
				map[Name:apiVersion Type:<nil> Value:apps/v1]
				map[Name:kind Type:<nil> Value:Deployment]
				map[Name:metadata.name Type:<nil> Value:http_dummy]
				map[Name:spec.template.spec.containers[0].image Type:<nil> Value:bigoyang/http_dummy:0.2.1]
				map[Name:spec.template.spec.containers[0].name Type:<nil> Value:http_dummy]
				map[Name:spec.selector.matchLabels.component Type:<nil> Value:http_dummy]
				map[Name:spec.template.metadata.labels.component Type:<nil> Value:http_dummy]
				map[Name:spec.template.spec.containers[0].env[0].name Type:string Value:Logging_LogLevel]
				map[Name:spec.template.spec.containers[0].env[0].value Type:<nil> Value:INFO]
				map[Name:spec.template.spec.containers[0].env[1].name Type:string Value:FLOGO_APP_PROPS_ENV]
				map[Name:spec.template.spec.containers[0].env[1].value Type:<nil> Value:auto]
				map[Name:spec.template.spec.containers[0].ports[0].name Type:String Value:9999]]]

		map[
			Group:ip-service
			Value:[
				map[Name:apiVersion Type:<nil> Value:v1]
				map[Name:kind Type:<nil> Value:Service]
				map[Name:metadata.name Type:<nil> Value:http_dummy-ip-service]
				map[Name:spec.selector.component Type:<nil> Value:http_dummy]
				map[Name:spec.type Type:<nil> Value:LoadBalancer]
				map[Name:spec.port[0] Type:String Value:8080]
				map[Name:spec.targetPort[0] Type:String Value:9999]
			]
		]]
	*/

	/* Get main configuration */
	f1Properties := f1PropertiesMaster[0].(map[string]interface{})["Value"].([]interface{})
	valueMap := make(map[string]interface{})
	typeMap := make(map[string]string)
	for _, prop := range airProperties {
		name := util.GetPropertyElementAsString("Name", prop)
		value := util.GetPropertyElement("Value", prop)
		dtype := util.GetPropertyElementAsString("Type", prop)
		valueMap[name] = value
		typeMap[name] = dtype
	}

	/*
		{ "Name" : "", }
	*/

	propIndex := 0
	for index, prop := range f1Properties {
		name := util.GetPropertyElementAsString("Name", prop)
		//dtype := util.GetPropertyElement("Type", prop)
		/* only find name part */
		if 0 == strings.Index(name, prefix) && ".name" == name[len(name)-5:] {
			/* air properties contains this name */
			value := util.GetPropertyElementAsString("Value", prop) // It's going to key so got to be string
			if nil != valueMap[value] {
				/* Next properties is the value so set it*/
				f1Properties[index+1].(map[string]interface{})["Value"] = valueMap[value] // ???????????????? "value"
				f1Properties[index+1].(map[string]interface{})["Type"] = typeMap[value]
				delete(valueMap, value)
			}
			propIndex++
		}
	}

	for name, value := range valueMap {
		f1Properties = append(f1Properties, map[string]interface{}{
			"Name":  fmt.Sprintf("%s[%d].name", prefix, propIndex),
			"Value": name,
			"Type":  "string",
		})
		f1Properties = append(f1Properties, map[string]interface{}{
			"Name":  fmt.Sprintf("%s[%d].value", prefix, propIndex),
			"Value": value,
			"Type":  typeMap[name],
		})
		propIndex++
	}

	for _, extraProperty := range extraProperties {
		property := extraProperty.(map[string]interface{})
		name := property["Name"].(string)
		if "App.LogLevel" == name {
			f1Properties = append(f1Properties, map[string]interface{}{
				"Name":  fmt.Sprintf("%s[%d].name", prefix, propIndex),
				"Value": "FLOGO_LOG_LEVEL",
				"Type":  "string",
			})
			f1Properties = append(f1Properties, map[string]interface{}{
				"Name":  fmt.Sprintf("%s[%d].value", prefix, propIndex),
				"Value": property["Value"],
				"Type":  "string",
			})
			propIndex++
		} else if strings.HasPrefix(name, "App.") {
			f1Properties = append(f1Properties, map[string]interface{}{
				"Name":  fmt.Sprintf("%s[%d].name", prefix, propIndex),
				"Value": strings.ReplaceAll(name, ".", "_"),
				"Type":  "string",
			})
			f1Properties = append(f1Properties, map[string]interface{}{
				"Name":  fmt.Sprintf("%s[%d].value", prefix, propIndex),
				"Value": property["Value"],
				"Type":  "string",
			})
			propIndex++
		}
	}

	/* Set main configuration back */
	f1PropertiesMaster[0].(map[string]interface{})["Value"] = f1Properties

	return f1PropertiesMaster
}

func createDockerF1Properties(
	prefix string,
	f1PropertiesMaster []interface{},
	airProperties []interface{},
	extraProperties []interface{},
) []interface{} {

	/* Get main configuration */
	f1Properties := f1PropertiesMaster[0].(map[string]interface{})["Value"].([]interface{})
	valueMap := make(map[string]interface{})
	typeMap := make(map[string]string)
	for _, prop := range airProperties {
		name := util.GetPropertyElementAsString("Name", prop)
		value := util.GetPropertyElement("Value", prop)
		dtype := util.GetPropertyElementAsString("Type", prop)
		valueMap[name] = value
		typeMap[name] = dtype
		log.Debug("name = ", name, ", value = ", value)
	}

	/*
		Update existing environment properties
	*/
	propIndex := 0
	for _, prop := range f1Properties {
		name := util.GetPropertyElementAsString("Name", prop)
		value := util.GetPropertyElementAsString("Value", prop) // this is docker env got to be string
		if 0 == strings.Index(name, prefix+"[") {
			pos := strings.Index(value, "=")
			key := value[0:pos]
			if nil != valueMap[key] {
				log.Debug("In update key = ", key, ", valueMap[key] = ", valueMap[key])
				prop.(map[string]interface{})["Value"] = fmt.Sprintf("%s=%s", key, valueMap[key])
				delete(valueMap, key)
			}
			propIndex++
		}
	}

	for name, value := range valueMap {
		log.Debug("In add key = ", name, ", valueMap[key] = ", value)
		f1Properties = append(f1Properties, map[string]interface{}{
			"Name":  fmt.Sprintf("%s[%d]", prefix, propIndex),
			"Value": fmt.Sprintf("%s=%s", name, value),
			"Type":  "string",
		})
		propIndex++
	}

	for _, extraProperty := range extraProperties {
		property := extraProperty.(map[string]interface{})
		name := property["Name"].(string)
		if "App.LogLevel" == name {
			f1Properties = append(f1Properties, map[string]interface{}{
				"Name":  fmt.Sprintf("%s[%d]", prefix, propIndex),
				"Value": fmt.Sprintf("FLOGO_LOG_LEVEL=%s", property["Value"]),
				"Type":  "string",
			})
			propIndex++
		} else if strings.HasPrefix(name, "App.") {
			f1Properties = append(f1Properties, map[string]interface{}{
				"Name":  fmt.Sprintf("%s[%d]", prefix, propIndex),
				"Value": fmt.Sprintf("%s=%s", strings.ReplaceAll(name, ".", "_"), property["Value"]),
				"Type":  "string",
			})
			propIndex++
		}
	}

	/* Set main configuration back */
	f1PropertiesMaster[0].(map[string]interface{})["Value"] = f1Properties

	return f1PropertiesMaster
}
