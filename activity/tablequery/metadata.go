package tablequery

import (
	"github.com/project-flogo/core/data/coerce"
	"github.com/project-flogo/core/support/connection"
)

type Settings struct {
	Table   connection.Manager `md:"Table,required"`
	Indices string             `md:"Indices,required"`
}

func (s *Settings) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Table":   s.Table,
		"Indices": s.Indices,
	}
}

// Input Structure
type Input struct {
	QueryKey map[string]interface{} `md:"QueryKey,required"`
}

// ToMap Input interface
func (i *Input) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"QueryKey": i.QueryKey,
	}
}

// FromMap Input interface
func (i *Input) FromMap(values map[string]interface{}) error {
	var err error
	i.QueryKey, err = coerce.ToObject(values["QueryKey"])
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
