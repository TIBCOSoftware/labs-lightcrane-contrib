package httpredirect

import (
	"github.com/project-flogo/core/data/coerce"
)

type Settings struct {
	Port string `md:"Port"`
}

type HandlerSettings struct {
	Path string `md:"Path"`
}

type Output struct {
	RequestURL string `md:"RequestURL"`
}

func (this *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"RequestURL": this.RequestURL,
	}
}

func (this *Output) FromMap(values map[string]interface{}) error {

	var err error
	this.RequestURL, err = coerce.ToString(values["RequestURL"])
	if err != nil {
		return err
	}

	return nil
}
