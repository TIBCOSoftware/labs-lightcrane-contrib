package httpfilehandler

import (
	"github.com/project-flogo/core/data/coerce"
)

type Settings struct {
	Port string `md:"Port"`
}

type HandlerSettings struct {
	Path       string `md:"Path"`
	BaseFolder string `md:"BaseFolder"`
}

type Output struct {
	Filename string `md:"Filename"`
	FilePath string `md:"FilePath"`
}

func (this *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Filename": this.Filename,
		"FilePath": this.FilePath,
	}
}

func (this *Output) FromMap(values map[string]interface{}) error {

	var err error
	this.Filename, err = coerce.ToString(values["Filename"])
	this.FilePath, err = coerce.ToString(values["FilePath"])
	if err != nil {
		return err
	}

	return nil
}
