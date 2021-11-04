package f1

import (
	"os"

	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/data/expression/function"
)

func init() {
	function.Register(&fnFolderExists{})
}

type fnFolderExists struct {
}

func (fnFolderExists) Name() string {
	return "folderexists"
}

func (fnFolderExists) Sig() (paramTypes []data.Type, isVariadic bool) {
	return []data.Type{data.TypeString}, false
}

func (fnFolderExists) Eval(params ...interface{}) (interface{}, error) {
	log.Info("FolderExists.eval] entering ..... ")
	defer log.Info("FolderExists.eval] exit ..... ")

	log.Info("FolderExists.eval] folder name = ", params[0])

	exist := true
	folderInfo, err := os.Stat(params[0].(string))

	if nil != err {
		log.Info("FolderExists.eval] err = ", err.Error())
	}

	if os.IsNotExist(err) {
		exist = false
	}

	log.Info("FolderExists.eval] folder info = ", folderInfo, ", exists = ", exist)

	return exist, nil
}
