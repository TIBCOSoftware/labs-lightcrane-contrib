package tablemutate

import (
	"github.com/project-flogo/core/data/coerce"
	"github.com/project-flogo/core/support/connection"
)

type Settings struct {
	Table  connection.Manager `md:"Table,required"`
	Method string             `md:"cacheSize,required"`
}

func (s *Settings) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Table":  s.Table,
		"Method": s.Method,
	}
}

// Input Structure
type Input struct {
	Mapping map[string]interface{} `md:"Mapping,required"`
}

// ToMap Input interface
func (i *Input) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Mapping": i.Mapping,
	}
}

// FromMap Input interface
func (i *Input) FromMap(values map[string]interface{}) error {
	var err error
	i.Mapping, err = coerce.ToObject(values["Mapping"])
	if err != nil {
		return err
	}

	return nil

}

//Output struct
type Output struct {
	Data   map[string]interface{} `md:"Data,required"`
	Exists bool                   `md:"Exists,required"`
}

// ToMap conversion
func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Data":   o.Data,
		"Exists": o.Exists,
	}
}

// FromMap conversion
func (o *Output) FromMap(values map[string]interface{}) error {
	var err error

	o.Data, err = coerce.ToObject(values["Data"])
	if err != nil {
		return err
	}

	o.Exists, err = coerce.ToBool(values["Exists"])
	if err != nil {
		return err
	}

	return nil
}
