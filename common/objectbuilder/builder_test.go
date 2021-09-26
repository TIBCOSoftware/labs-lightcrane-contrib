package objectbuilder

import (
	"fmt"
	"testing"
)

const (
	iPorts      = "ports"
	iProperties = "properties"
)

func TestBuildTemplates(t *testing.T) {
	result := StringReplacement(
		map[string]interface{}{
			"A": map[string]interface{}{
				"B": "=$property[\"Logging.LogLevel\"]",
			},
			"C": "MQTTTrigger.MaximumQOS",
		},
		map[string]interface{}{
			"=$property[\"Logging.LogLevel\"]": "=$property[\"DataSource.Logging.LogLevel\"]",
			"MQTTTrigger.MaximumQOS":           "DataSource.MQTTTrigger.MaximumQOS",
		})

	fmt.Println("-------", result)
}
