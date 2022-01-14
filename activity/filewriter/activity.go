/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package filewriter

import (
	"archive/zip"
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	kwr "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/keywordreplace"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

const (
	sInputType        = "inputType"
	sOutputFile       = "outputFile"
	sLeftToken        = "leftToken"
	sRightToken       = "rightToken"
	sVariablesDef     = "variablesDef"
	iData             = "Data"
	iInput            = "Input"
	iFilePathVariable = "Variables"
	iSkipCondition    = "SkipCondition"
	oFilename         = "Filename"
	oVariablesOut     = "VariablesOut"
)

var log = logger.GetLogger("tibco-activity-filewriter")

type FileWriterActivity struct {
	metadata    *activity.Metadata
	pathMappers map[string]*kwr.KeywordMapper
	variables   map[string]map[string]string
	mux         sync.Mutex
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	return &FileWriterActivity{
		metadata:    metadata,
		pathMappers: make(map[string]*kwr.KeywordMapper),
		variables:   make(map[string]map[string]string),
	}
}

func (a *FileWriterActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *FileWriterActivity) Eval(context activity.Context) (done bool, err error) {
	log.Debug("(FileWriterActivity.Eval) Graph to file entering ......... ")
	defer log.Debug("(FileWriterActivity.Eval) write object to file exit ......... ")

	skipCondition := context.GetInput(iSkipCondition).(bool)
	if skipCondition {
		log.Debug("(Eval) Skip taks : ", skipCondition)
		context.SetOutput(oFilename, nil)
		return true, nil
	}

	data := context.GetInput(iData)
	inputType, _ := context.GetSetting(sInputType)

	var outputFile string
	pathVariable := context.GetInput(iFilePathVariable)
	if nil != pathVariable {
		pathMapper, _, _ := a.getPathMapper(context)
		outputFile = pathMapper.Replace("", pathVariable.(map[string]interface{}))
	}

	a.prepareFolder(outputFile)
	if strings.HasSuffix(strings.ToLower(outputFile), ".zip.base64") || strings.HasSuffix(strings.ToLower(outputFile), ".zip") {
		a.handelZipFile(outputFile, data)
	} else {
		a.handelFile(outputFile, data, inputType)
	}

	context.SetOutput(oFilename, outputFile)
	context.SetOutput(oVariablesOut, pathVariable)

	return true, nil
}

func (a *FileWriterActivity) getPathMapper(ctx activity.Context) (*kwr.KeywordMapper, map[string]string, error) {
	myId := util.ActivityId(ctx)
	mapper := a.pathMappers[myId]
	variables := a.variables[myId]

	if nil == mapper {
		a.mux.Lock()
		defer a.mux.Unlock()
		mapper = a.pathMappers[myId]
		if nil == mapper {
			variables = make(map[string]string)
			variablesDef, _ := ctx.GetSetting(sVariablesDef)
			log.Debug("Processing handlers : variablesDef = ", variablesDef)
			for _, variableDef := range variablesDef.([]interface{}) {
				variableInfo := variableDef.(map[string]interface{})
				variables[variableInfo["Name"].(string)] = variableInfo["Type"].(string)
			}

			outputFile, exist := ctx.GetSetting(sOutputFile)
			if !exist {
				return nil, nil, errors.New("Output file path not defined!")
			}

			lefttoken, exist := ctx.GetSetting(sLeftToken)
			if !exist {
				return nil, nil, errors.New("LeftToken not defined!")
			}
			righttoken, exist := ctx.GetSetting(sRightToken)
			if !exist {
				return nil, nil, errors.New("RightToken not defined!")
			}
			mapper = kwr.NewKeywordMapper(outputFile.(string), lefttoken.(string), righttoken.(string))

			a.pathMappers[myId] = mapper
			a.variables[myId] = variables
		}
	}
	return mapper, variables, nil
}

func (a *FileWriterActivity) prepareFolder(outputFile string) error {
	outputFolder := filepath.Dir(outputFile)

	log.Debug("Output file : ", outputFile)
	log.Debug("Output folder : ", outputFolder)

	// Check if folder exists
	_, err := os.Stat(outputFolder)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(outputFolder, os.ModePerm)
			if nil != err {
				log.Error("Unable to create folder : ", err)
				return err
			}
		}
	}

	log.Debug("Initializing FileWriter Service end ...")
	return nil
}

func (a *FileWriterActivity) handelFile(outputFile string, dataEnvelop interface{}, inputType interface{}) error {
	var dataString string
	if "String" == inputType {
		dataString = dataEnvelop.(map[string]interface{})[iInput].(string)
	} else {
		jsonString, _ := json.Marshal(dataEnvelop)
		dataString = string(jsonString) + "\r\n"
	}

	var err error
	var dataBytes []byte
	if strings.HasSuffix(strings.ToLower(outputFile), ".base64") {
		dataBytes, err = b64.StdEncoding.DecodeString(dataString)
		dataString = string(dataBytes)
	}

	a.mux.Lock()
	defer a.mux.Unlock()

	log.Debug("(Eval) File name : ", outputFile)
	f, err := os.OpenFile(outputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		panic(err)
		return err
	}

	defer f.Close()

	f.WriteString(dataString)
	return nil
}

func (a *FileWriterActivity) handelZipFile(fullFilename string, dataEnvelop interface{}) error {
	b64data := dataEnvelop.(map[string]interface{})[iInput].(string)
	data, err := b64.StdEncoding.DecodeString(string(b64data))

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		log.Error(err)
	}

	folder := filepath.Dir(fullFilename)
	for _, f := range zipReader.File {
		log.Debug("processing file ", f.Name)
		filePath := filepath.Join(folder, f.Name)
		if f.FileInfo().IsDir() {
			log.Debug("creating directory...")
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		log.Debug("unzipping file ", filePath)
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
	return nil
}
