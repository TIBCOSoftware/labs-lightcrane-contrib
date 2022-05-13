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

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
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
	log.Debug("(FileReaderActivity.Eval) Entering ......... ")
	defer log.Debug("(FileReaderActivity.Eval) Exit ......... ")

	baseFolder, err := a.getBaseFolder(context)
	if nil != err {
		return false, err
	}
	log.Debug("(FileReaderActivity.Eval) Output base folder = ", baseFolder)

	a.mux.Lock()
	defer a.mux.Unlock()

	filePattern := context.GetInput(iFilePattern).(string)

	if "" != baseFolder {
		filePattern = fmt.Sprintf("%s/%s", baseFolder, filePattern)
	}
	log.Debug("(FileReaderActivity.Eval) filePattern = ", filePattern)

	results := make([]map[string]interface{}, 0)
	content, err := readFile(filePattern)
	if nil != err {
		log.Warn("(FileReaderActivity.Eval) err : ", err)
		matches, err := filepath.Glob(filePattern)
		if nil != err {
			log.Warn("(FileReaderActivity.Eval) err : ", err)
			return false, err
		}

		log.Debug("(FileReaderActivity.Eval) File pattern : ", filePattern, ", matches : ", matches)

		for _, filename := range matches {
			content, err := readFile(filename)
			if nil != err {
				log.Warn("(FileReaderActivity.Eval) err : ", err)
				continue
			}
			results = append(results, map[string]interface{}{"Filename": filename, "Content": content})
		}
	} else {
		results = append(results, map[string]interface{}{"Filename": filePattern, "Content": content})
	}

	//log.Debug("(FileReaderActivity.Eval) results : ", results)
	context.SetOutput(oResults, results)

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
			log.Debug("Initializing FileReader Service start ...")

			baseFolderSetting, _ := context.GetSetting(sBaseFolder)
			baseFolder = baseFolderSetting.(string)

			log.Debug("Initializing FileReader Service end ...")
			a.baseFolders[myId] = baseFolder
		}
	}

	return baseFolder, nil
}

func readFile(filename string) (string, error) {
	//log.Debug("(FileReaderActivity.readFile) filename = ", filename)
	fileContent, err := ioutil.ReadFile(filename)
	//log.Debug("(FileReaderActivity.readFile) fileContent = ", fileContent)
	if err != nil {
		return "", err
	}
	return string(fileContent), nil
}
