/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package exec

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/exec/execeventbroker"
	kwr "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/keywordreplace"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

var log = logger.GetLogger("tibco-f1-exec")

var initialized bool = false

const (
	cConnection      = "execConnection"
	cConnectionName  = "name"
	sWorkingFolder   = "workingFolder"
	sNumOfExecutions = "numOfExecutions"
	sLeftToken       = "leftToken"
	sRightToken      = "rightToken"
	sVariablesDef    = "variablesDef"
	sSystemEnv       = "SystemEnv"
	iExecutable      = "Executable"
	iExecutions      = "Executions"
	iExecution       = "Execution"
	iSystemEnvs      = "SystemEnvs"
	iAsynchronous    = "Asynchronous"
	iVariable        = "Variables"
	iSkipCondition   = "SkipCondition"
	oSuccess         = "Success"
	oData            = "Data"
	oMessage         = "Message"
	oErrorCode       = "ErrorCode"
	oResult          = "Result"
)

type ExecActivity struct {
	metadata            *activity.Metadata
	mux                 sync.Mutex
	pathMappers         map[string]*kwr.KeywordMapper
	variables           map[string]map[string]string
	sysEnvs             map[string]map[string]string
	activityToConnector map[string]string
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aExecActivity := &ExecActivity{
		metadata:            metadata,
		pathMappers:         make(map[string]*kwr.KeywordMapper),
		variables:           make(map[string]map[string]string),
		sysEnvs:             make(map[string]map[string]string),
		activityToConnector: make(map[string]string),
	}

	return aExecActivity
}

func (a *ExecActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *ExecActivity) Eval(context activity.Context) (done bool, err error) {

	log.Debug("[ExecActivity.Eval] entering ........ ")
	defer log.Debug("[ExecActivity.Eval] Exit ........ ")

	skipCondition := context.GetInput(iSkipCondition).(bool)
	if skipCondition {
		log.Debug("(ExecActivity.Eval) Skip taks : ", skipCondition)
		success := true
		message := "Command Skiped!"
		context.SetOutput(oSuccess, success)
		context.SetOutput(oMessage, message)
		context.SetOutput(oErrorCode, 100)
		context.SetOutput(oResult, make(map[string]interface{}))
		return true, nil
	}

	iAsynchronous, ok := context.GetInput(iAsynchronous).(bool)
	if !ok {
		iAsynchronous = false
	}

	sysEnv, err := a.getSysEnvs(context)
	if nil != err {
		log.Debug("(ExecActivity.Eval) Unable to load sysEnv : ", err.Error())
		sysEnv = make(map[string]string)
	}

	executable, ok := context.GetInput(iExecutable).(map[string]interface{})
	if !ok {
		return false, errors.New("Invalid executable ... ")
	}

	var dynSysEnvs map[string]interface{}
	if nil != executable[iSystemEnvs] {
		dynSysEnvs = executable[iSystemEnvs].(map[string]interface{})
	} else {
		dynSysEnvs = make(map[string]interface{})
	}

	defVariable := context.GetInput(iVariable).(map[string]interface{})
	variable := make(map[string]interface{})
	for key, value := range defVariable {
		if nil != dynSysEnvs[key] {
			variable[key] = dynSysEnvs[key]
		} else {
			variable[key] = value
		}
	}
	executions := executable[iExecutions].(map[string]interface{})
	numOfExecutions, _ := context.GetSetting(sNumOfExecutions)
	pathMapper, _, _ := a.getVariableMapper(context)
	var commands [][]string
	for i := 0; i < numOfExecutions.(int); i++ {
		if nil != variable {
			command := pathMapper.Replace(executions[fmt.Sprintf("%s_%d", iExecution, i)].(string), variable)
			log.Debug("(ExecActivity.Eval) command : ", command)
			commands = append(commands, strings.Split(command, " "))
		}
	}

	workingFolder, exist := context.GetSetting(sWorkingFolder)
	if exist {
		workingFolder = pathMapper.Replace(workingFolder.(string), variable)
	}

	newEnv := os.Environ()
	for key, value := range dynSysEnvs {
		newEnv = append(newEnv, fmt.Sprintf("%s=%s", key, value))
	}
	for key, value := range sysEnv {
		newEnv = append(newEnv, fmt.Sprintf("%s=%s", key, value))
	}
	log.Debug("[ExecActivity.Eval] newEnv : ", newEnv)

	log.Debug("(ExecActivity.Eval) iAsynchronous : ", iAsynchronous)
	eventListener, _ := a.getExecEventBroker(context)
	execContext := map[string]interface{}{
		"Variable":          variable,
		"SystemEnvironment": dynSysEnvs,
		"Successful":        true,
		"ErrorMsg":          "",
	}
	data := make(map[string]interface{})
	if iAsynchronous {
		log.Debug("(ExecActivity.Eval) execCommand asynchronously!")
		go a.execCommand(commands, newEnv, workingFolder, execContext, eventListener)
	} else {
		log.Debug("(ExecActivity.Eval) execCommand synchronously!")
		data, err = a.execCommand(commands, newEnv, workingFolder, execContext, eventListener)
	}

	success := true
	message := "Command Executed!"
	if nil != err {
		success = false
		message = err.Error()
	}

	context.SetOutput(oSuccess, success)
	context.SetOutput(oMessage, message)
	context.SetOutput(oErrorCode, 100)
	context.SetOutput(oResult, data["Result"])

	return true, nil
}

func (a *ExecActivity) execCommand(
	commands [][]string,
	newEnv []string,
	workingFolder interface{},
	execContext map[string]interface{},
	listener *execeventbroker.EXEEventBroker) (map[string]interface{}, error) {
	log.Debug("[ExecActivity.execCommand] entering - execContext : ", execContext)
	var err error
	errorMsgs := make([]interface{}, 0)
	data := make(map[string]interface{})
	data["Result"] = make([]interface{}, 0)

	for i := 0; i < len(commands); i++ {
		log.Debug("[ExecActivity.execCommand] command : ", commands)
		cmd := exec.Command(commands[i][0], commands[i][1:]...)
		if nil != workingFolder {
			log.Debug("[ExecActivity.execCommand] Working folder : ", workingFolder.(string))
			cmd.Dir = workingFolder.(string)
			_, err := os.Stat(workingFolder.(string))
			if err != nil {
				if os.IsNotExist(err) {
					err := os.MkdirAll(workingFolder.(string), os.ModePerm)
					if nil != err {
						log.Error("[ExecActivity.execCommand] Unable to create folder : ", err)
						return data, err
					}
				}
			}
		}

		cmd.Env = newEnv
		for _, env := range cmd.Env {
			log.Debug("[ExecActivity.execCommand] ", env)
		}

		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

		err = cmd.Run()
		if err != nil {
			log.Errorf("[ExecActivity.execCommand] failed with %s\n", err)
			errorMsgs = append(errorMsgs, err.Error())
			break
		}
		data["Result"] = append(data["Result"].([]interface{}), map[string]interface{}{
			"Command": commands[i][0],
			"StdOut":  string(stdoutBuf.Bytes()),
			"StdErr":  string(stderrBuf.Bytes()),
		})
	}
	log.Debug("[ExecActivity.execCommand] return - data : ", data)
	if nil != listener {
		if nil != err {
			execContext["Successful"] = false
			execContext["ErrorMsg"] = errorMsgs
		}
		log.Debug("[ExecActivity.execCommand] send event - execContext : ", execContext)
		listener.SendEvent(execContext)
	}
	return data, err
}

func (a *ExecActivity) getSysEnvs(ctx activity.Context) (map[string]string, error) {
	myId := util.ActivityId(ctx)
	sysEnvs := a.sysEnvs[myId]

	if nil == sysEnvs {
		a.mux.Lock()
		defer a.mux.Unlock()
		sysEnvs = a.sysEnvs[myId]
		if nil == sysEnvs {
			log.Debug("[ExecActivity.getSysEnvs] activity.Context = ", ctx)
			sysEnvs = make(map[string]string)
			sysEnvsDef, _ := ctx.GetSetting(sSystemEnv)
			log.Debug("[ExecActivity.getSysEnvs] sysEnvsDef = ", sysEnvsDef)
			if nil != sysEnvsDef {
				for _, sysEnvDef := range sysEnvsDef.([]interface{}) {
					sysEnvInfo := sysEnvDef.(map[string]interface{})
					if "No" == sysEnvInfo["PerCommand"] {
						sysEnvs[sysEnvInfo["Key"].(string)] = sysEnvInfo["Value"].(string)
					}
				}
			}
			log.Debug("[ExecActivity.getSysEnvs] sysEnvs = ", sysEnvs)

			a.sysEnvs[myId] = sysEnvs
		}
	}
	return sysEnvs, nil
}

func (a *ExecActivity) getVariableMapper(ctx activity.Context) (*kwr.KeywordMapper, map[string]string, error) {
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
			log.Debug("ExecActivity.Processing handlers : variablesDef = ", variablesDef)
			for _, variableDef := range variablesDef.([]interface{}) {
				variableInfo := variableDef.(map[string]interface{})
				variables[variableInfo["Name"].(string)] = variableInfo["Type"].(string)
			}

			lefttoken, exist := ctx.GetSetting(sLeftToken)
			if !exist {
				return nil, nil, errors.New("LeftToken not defined!")
			}
			righttoken, exist := ctx.GetSetting(sRightToken)
			if !exist {
				return nil, nil, errors.New("RightToken not defined!")
			}
			mapper = kwr.NewKeywordMapper("", lefttoken.(string), righttoken.(string))

			a.pathMappers[myId] = mapper
			a.variables[myId] = variables
		}
	}
	return mapper, variables, nil
}

func (a *ExecActivity) getExecEventBroker(context activity.Context) (*execeventbroker.EXEEventBroker, error) {
	myId := util.ActivityId(context)

	exeEventBroker := execeventbroker.GetFactory().GetEXEEventBroker(a.activityToConnector[myId])
	if nil == exeEventBroker {
		log.Debug("Look up ececution event broker start ...")
		connection, exist := context.GetSetting(cConnection)
		if !exist {
			log.Warn("Execution event broker not configured! ")
			return nil, nil
		}

		connectionInfo, _ := data.CoerceToObject(connection)
		if connectionInfo == nil {
			return nil, activity.NewError("Execution event connection not able to be parsed", "TGDB-SSE-4001", nil)
		}

		var connectorName string
		connectionSettings, _ := connectionInfo["settings"].([]interface{})
		if connectionSettings != nil {
			for _, v := range connectionSettings {
				setting, _ := data.CoerceToObject(v)
				if setting != nil {
					if setting["name"] == cConnectionName {
						connectorName, _ = data.CoerceToString(setting["value"])
					}
				}
			}
			exeEventBroker = execeventbroker.GetFactory().GetEXEEventBroker(connectorName)
			if nil == exeEventBroker {
				return nil, activity.NewError("Execution event broker not found, connection id = "+connectorName, "TGDB-SSE-4002", nil)
			}
			a.activityToConnector[myId] = connectorName
		}
		log.Debug("Look up SSE data broker end ...")
	}

	return exeEventBroker, nil
}
