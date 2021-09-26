/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package filereader

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/util"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
)

const (
	sBaseFolder  = "BaseFolder"
	iFilePattern = "FilePattern"
	oResults     = "Results"
)

var log = logger.GetLogger("tibco-activity-filereader")

type FileReaderActivity struct {
	metadata    *activity.Metadata
	baseFolders map[string]string
	mux         sync.Mutex
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	return &FileReaderActivity{
		metadata:    metadata,
		baseFolders: make(map[string]string),
	}
}

func (a *FileReaderActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *FileReaderActivity) Eval(context activity.Context) (done bool, err error) {
	log.Info("(FileReaderActivity.Eval) Entering ......... ")

	baseFolder, err := a.getBaseFolder(context)
	if nil != err {
		return false, err
	}
	log.Info("(FileReaderActivity.Eval) Output base folder = ", baseFolder)

	a.mux.Lock()
	defer a.mux.Unlock()

	filePattern := context.GetInput(iFilePattern).(string)

	if "" != baseFolder {
		filePattern = fmt.Sprintf("%s/%s", baseFolder, filePattern)
	}

	matches, err := filepath.Glob(filePattern)

	log.Info("(FileReaderActivity.Eval) File pattern : ", filePattern)

	results := make([]map[string]interface{}, 0)
	for _, filename := range matches {
		content, err := readFile(filename)
		if nil != err {
			continue
		}
		results = append(results, map[string]interface{}{"Filename": filename, "Content": content})
	}
	context.SetOutput(oResults, results)

	log.Info("(FileReaderActivity.Eval) Exit ......... ")
	return true, nil
}

func (a *FileReaderActivity) getBaseFolder(context activity.Context) (string, error) {

	myId := util.ActivityId(context)
	baseFolder := a.baseFolders[myId]

	if "" == baseFolder {
		a.mux.Lock()
		defer a.mux.Unlock()
		baseFolder = a.baseFolders[myId]
		if "" == baseFolder {
			log.Info("Initializing FileReader Service start ...")

			baseFolderSetting, _ := context.GetSetting(sBaseFolder)
			baseFolder = baseFolderSetting.(string)

			log.Info("Initializing FileReader Service end ...")
			a.baseFolders[myId] = baseFolder
		}
	}

	return baseFolder, nil
}

func readFile(filename string) (string, error) {
	fileContent, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("File reading error", err)
		return "", err
	}
	return string(fileContent), nil
}
