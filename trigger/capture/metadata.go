package capture

import (
	"github.com/project-flogo/core/support/connection"
)

type Settings struct {
}

type HandlerSettings struct {
	Connection connection.Manager `md:"execConnection,required"`
}

type Output struct {
	Event map[string]interface{} `md:"Event"`
}

func (this *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Event": this.Event,
	}
}

func (this *Output) FromMap(values map[string]interface{}) error {

	this.Event = values["Event"].(map[string]interface{})

	return nil
}
