/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package filewriter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
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
	pathMappers map[string]*KeywordMapper
	variables   map[string]map[string]string
	mux         sync.Mutex
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	return &FileWriterActivity{
		metadata:    metadata,
		pathMappers: make(map[string]*KeywordMapper),
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

	var dataString string
	data := context.GetInput(iData)
	inputType, _ := context.GetSetting(sInputType)
	if "String" == inputType {
		dataString = data.(map[string]interface{})[iInput].(string)
	} else {
		jsonString, _ := json.Marshal(data)
		dataString = string(jsonString) + "\r\n"
	}

	pathVariable := context.GetInput(iFilePathVariable)

	var outputFile string
	if nil != pathVariable {
		pathMapper, _, _ := a.getPathMapper(context)
		outputFile = pathMapper.replace("", pathVariable.(map[string]interface{}))
	}

	a.mux.Lock()
	defer a.mux.Unlock()

	err = a.prepareFile(outputFile)
	if err != nil {
		panic(err)
		return false, nil
	}

	log.Debug("(Eval) File name : ", outputFile)
	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
		return false, nil
	}

	defer f.Close()

	f.WriteString(dataString)

	log.Debug("(Eval) list files in : ", filepath.Dir(outputFile))
	files, err := ioutil.ReadDir(filepath.Dir(outputFile))
	if err != nil {
		log.Error(err)
	}

	for _, f := range files {
		log.Debug("(Eval) file : ", f.Name())
	}

	context.SetOutput(oFilename, outputFile)
	context.SetOutput(oVariablesOut, pathVariable)

	return true, nil
}

func (a *FileWriterActivity) getPathMapper(ctx activity.Context) (*KeywordMapper, map[string]string, error) {
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
			mapper = NewKeywordMapper(outputFile.(string), lefttoken.(string), righttoken.(string))

			a.pathMappers[myId] = mapper
			a.variables[myId] = variables
		}
	}
	return mapper, variables, nil
}

func (a *FileWriterActivity) prepareFile(outputFile string) error {

	fileExist := true
	_, err := os.Stat(outputFile)
	if nil != err {
		if os.IsNotExist(err) {
			fileExist = false
		}
	}

	if !fileExist {
		outputFolder := filepath.Dir(outputFile)

		log.Debug("Output file : ", outputFile)
		log.Debug("Output folder : ", outputFolder)

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

		_, err = os.Create(outputFile)
		if nil != err {
			log.Error("Unable to create file : ", err)
			return err
		}
	} else {
		if true {
			err = os.Remove(outputFile)
			if nil != err {
				log.Error("Unable to create file : ", err)
			}

			_, err = os.Create(outputFile)
			if nil != err {
				log.Error("Unable to create file : ", err)
				return err
			}
		}
	}

	log.Debug("Initializing FileWriter Service end ...")

	return nil
}

type KeywordReplaceHandler struct {
	result     string
	keywordMap map[string]interface{}
}

func (this *KeywordReplaceHandler) setMap(keywordMap map[string]interface{}) {
	this.keywordMap = keywordMap
}

func (this *KeywordReplaceHandler) startToMap() {
	this.result = ""
}

func (this *KeywordReplaceHandler) replace(keyword string) string {
	if nil != this.keywordMap[keyword] {
		return this.keywordMap[keyword].(string)
	}
	return ""
}

func (this *KeywordReplaceHandler) endOfMapping(document string) {
	this.result = document
}

func (this *KeywordReplaceHandler) getResult() string {
	return this.result
}

func NewKeywordMapper(
	template string,
	lefttag string,
	righttag string) *KeywordMapper {
	mapper := KeywordMapper{
		template:     template,
		keywordOnly:  false,
		slefttag:     lefttag,
		srighttag:    righttag,
		slefttaglen:  len(lefttag),
		srighttaglen: len(righttag),
	}
	return &mapper
}

type KeywordMapper struct {
	template     string
	keywordOnly  bool
	slefttag     string
	srighttag    string
	slefttaglen  int
	srighttaglen int
	document     bytes.Buffer
	keyword      bytes.Buffer
	mh           KeywordReplaceHandler
}

func (this *KeywordMapper) replace(template string, keywordMap map[string]interface{}) string {
	if "" == template {
		template = this.template
		if "" == template {
			return ""
		}
	}

	log.Debug("[KeywordMapper.replace] template = ", template)

	this.mh.setMap(keywordMap)
	this.document.Reset()
	this.keyword.Reset()

	scope := false
	boundary := false
	skeyword := ""
	svalue := ""

	this.mh.startToMap()
	for i := 0; i < len(template); i++ {
		//log.Debugf("template[%d] = ", i, template[i])
		// maybe find a keyword beginning Tag - now isn't in a keyword
		if !scope && template[i] == this.slefttag[0] {
			if this.isATag(i, this.slefttag, template) {
				this.keyword.Reset()
				scope = true
			}
		} else if scope && template[i] == this.srighttag[0] {
			// maybe find a keyword ending Tag - now in a keyword
			if this.isATag(i, this.srighttag, template) {
				i = i + this.srighttaglen - 1
				skeyword = this.keyword.String()[this.slefttaglen:this.keyword.Len()]
				svalue = this.mh.replace(skeyword)
				if "" == svalue {
					svalue = fmt.Sprintf("%s%s%s", this.slefttag, skeyword, this.srighttag)
				}
				//log.Debug("value ->", svalue);
				this.document.WriteString(svalue)
				boundary = true
				scope = false
			}
		}

		if !boundary {
			if !scope && !this.keywordOnly {
				this.document.WriteByte(template[i])
			} else {
				this.keyword.WriteByte(template[i])
			}
		} else {
			boundary = false
		}

		//log.Debug("document = ", this.document)

	}
	this.mh.endOfMapping(this.document.String())
	return this.mh.getResult()
}

func (this *KeywordMapper) isATag(i int, tag string, template string) bool {
	for j := 0; j < len(tag); j++ {
		if tag[j] != template[i+j] {
			return false
		}
	}
	return true
}

/*
  public static void main(String[] argc)
  {
     KeywordMapper ikm = new KeywordMapper();
     //ikm.setMapping("KEY1", "REAL_KEY1");
     //ikm.setMapping("KEY2", "REAL_KEY2");
     //ikm.setMapping("KEY3", "REAL_KEY3");
     //ikm.setLeftTag("%");
     //ikm.setRightTag("%");
     System.out.println("Result ---> " + ikm.replace("parameter1=$KEY1$,parameter2=$KEY2$,parameter3=$KEY3$"));
  }
*/
