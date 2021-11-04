package execlistener

type Settings struct {
}

type HandlerSettings struct {
	EventBoker string `md:"eventBoker,required"` // Execution Event Broker
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
