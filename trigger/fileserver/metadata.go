package fileserver

import (
	"github.com/project-flogo/core/data/coerce"
)

type Settings struct {
	Port      string `md:"Port"`
	EnableTLS bool   `md:"enableTLS"` // Enable TLS on the server
	CertFile  string `md:"certFile"`  // The path to PEM encoded server certificate
	KeyFile   string `md:"keyFile"`   // The path to PEM encoded server key
}

type HandlerSettings struct {
	Dir     string `md:"Dir"`
	URLPath string `md:"URLPath"`
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
