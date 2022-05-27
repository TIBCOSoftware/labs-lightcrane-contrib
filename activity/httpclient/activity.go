/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package httpclient

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	kwr "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/keywordreplace"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

var log = logger.GetLogger("tibco-model-ops-httpclient")

var initialized bool = false

const (
	sMethod        = "method"
	sTimeout       = "timeout"
	sLeftToken     = "leftToken"
	sRightToken    = "rightToken"
	sUrlMapping    = "urlMapping"
	sVariablesDef  = "variablesDef"
	sHttpHeaders   = "httpHeaders"
	iURL           = "URL"
	iHeaders       = "Headers"
	iMethod        = "Method"
	iBody          = "Body"
	iVariable      = "Variables"
	iSkipCondition = "SkipCondition"
	oSuccess       = "Success"
	oData          = "Data"
	oErrorCode     = "ErrorCode"
)

type HTTPClientActivity struct {
	metadata    *activity.Metadata
	mux         sync.Mutex
	urlMappers  map[string]map[string]string
	pathMappers map[string]*kwr.KeywordMapper
	variables   map[string]map[string]string
	header      map[string]map[string]string
}

func NewActivity(metadata *activity.Metadata) activity.Activity {
	aHTTPClientActivity := &HTTPClientActivity{
		metadata:    metadata,
		urlMappers:  make(map[string]map[string]string),
		pathMappers: make(map[string]*kwr.KeywordMapper),
		variables:   make(map[string]map[string]string),
		header:      make(map[string]map[string]string),
	}

	return aHTTPClientActivity
}

func (a *HTTPClientActivity) Metadata() *activity.Metadata {
	return a.metadata
}

func (a *HTTPClientActivity) Eval(context activity.Context) (done bool, err error) {

	log.Debug("[HTTPClientActivity:Eval] entering ........ ")
	defer log.Debug("[HTTPClientActivity:Eval] Exit ........ ")

	skipCondition := context.GetInput(iSkipCondition).(bool)
	if skipCondition {
		log.Debug("[HTTPClientActivity:Eval] Skip taks : ", skipCondition)
		return true, nil
	}

	url, ok := context.GetInput(iURL).(string)
	if !ok {
		return false, errors.New("Invalid request ... ")
	}

	urlMapper, _ := a.getURLMapper(context)
	if 0 == len(urlMapper) {
		variable := context.GetInput(iVariable)
		if nil != variable {
			pathMapper, _, _ := a.getVariableMapper(context)
			url = pathMapper.Replace(url, variable.(map[string]interface{}))
		}
	} else {
		url = urlMapper[url]
	}

	log.Debug("[HTTPClientActivity:Eval] url : ", url)

	var success bool
	var errorCode int
	var data string
	statusCode := 600
	if "" != url {
		var method interface{}
		runtimeMethod, ok := context.GetInput(iMethod).(string)
		log.Debug("[HTTPClientActivity:Eval] Runtime method : ", runtimeMethod)
		if !ok || "" == runtimeMethod {
			method, ok = context.GetSetting(sMethod)
			if !ok {
				return false, errors.New("Query method not defined!")
			}
			log.Debug("[HTTPClientActivity:Eval] Default method : ", method)
		} else {
			method = runtimeMethod
		}
		log.Debug("[HTTPClientActivity:Eval] Query method : ", method)

		var header map[string]string
		runtimeHeaders, ok := context.GetInput(iHeaders).([]interface{})
		log.Debug("[HTTPClientActivity:Eval] Runtime headers : ", runtimeHeaders)
		if !ok || nil == runtimeHeaders {
			header, err = a.getHeader(context)
			if nil != err {
				return false, errors.New("Invalid headers ... ")
			}
		} else {
			header = make(map[string]string)
			for _, runtimeHeader := range runtimeHeaders {
				headerInfo := runtimeHeader.(map[string]interface{})
				if nil != headerInfo["Key"] {
					header[headerInfo["Key"].(string)] = headerInfo["Value"].(string)
				} else {
					header[headerInfo["key"].(string)] = headerInfo["value"].(string)
				}
			}
		}

		timeout := time.Millisecond * time.Duration(10000)
		t, exist := context.GetSetting(sTimeout)
		if exist {
			timeout = time.Millisecond * time.Duration(t.(int))
		}

		var reqBody []byte
		var body []byte
		if "GET" == method.(string) {
			body, statusCode, err = a.get(url, header, timeout)
		} else if "DELETE" == method.(string) {
			body, statusCode, err = a.delete(url, header, timeout)
		} else if "POST" == method.(string) {
			if inBody, ok := context.GetInput(iBody).(string); ok {
				reqBody = []byte(inBody)
			} else if inBody, ok := context.GetInput(iBody).([]byte); ok {
				reqBody = inBody
			} else {
				return false, errors.New("Invalid request body ... ")
			}
			body, statusCode, err = a.post(url, header, timeout, (reqBody))
		} else if "PUT" == method.(string) {
			if inBody, ok := context.GetInput(iBody).(string); ok {
				reqBody = []byte(inBody)
			} else if inBody, ok := context.GetInput(iBody).([]byte); ok {
				reqBody = inBody
			} else {
				return false, errors.New("Invalid request body ... ")
			}
			body, statusCode, err = a.put(url, header, timeout, []byte(reqBody))
		} else {
			return false, errors.New("Query method not support!")
		}
		if nil != err {
			log.Debug("[HTTPClientActivity:Eval] Error : ", err.Error())
			success = false
			data = fmt.Sprintf("{\"Error\" : %s}", err.Error())
			errorCode = 300
		} else {
			success = true
			data = string(body)
			errorCode = 100
		}
	} else {
		log.Error("[HTTPClientActivity:Eval] Error : No URL defined!")
		success = false
		data = "{\"Error\" : \"No URL defined!\"}"
		errorCode = 300
	}

	if 200 != statusCode {
		success = false
	}
	context.SetOutput(oSuccess, success)
	context.SetOutput(oData, data)
	context.SetOutput(oErrorCode, errorCode)
	log.Debug("[HTTPClientActivity:Eval] success : ", success)

	return true, nil
}

func (a *HTTPClientActivity) get(url string, header map[string]string, timeout time.Duration) ([]byte, int, error) {
	log.Debug("[HTTPClientActivity:get] request url = ", url)
	defer log.Debug("[HTTPClientActivity:get] exit ... ")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error("[HTTPClientActivity:get] Error reading request. ", err)
		return nil, 500, err
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Error("[HTTPClientActivity:get] Error reading response. ", err)
		return nil, 500, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("[HTTPClientActivity:get] Error reading body. ", err)
		return nil, 500, err
	}
	defer log.Debug("[HTTPClientActivity:get] response body = ", string(body))

	return body, resp.StatusCode, nil
}

func (a *HTTPClientActivity) delete(url string, header map[string]string, timeout time.Duration) ([]byte, int, error) {
	log.Debug("[HTTPClientActivity:delete] enter, request url = ", url)
	defer log.Debug("[HTTPClientActivity:delete] exit ... ")

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Error("[HTTPClientActivity:get] Error reading request. ", err)
		return nil, 500, err
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Error("[HTTPClientActivity:delete] Error reading response. ", err)
		return nil, 500, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("[HTTPClientActivity:delete] Error reading body. ", err)
		return nil, 500, err
	}
	log.Debug("[HTTPClientActivity:delete] response body = ", string(body))

	return body, resp.StatusCode, nil
}

func (a *HTTPClientActivity) post(url string, header map[string]string, timeout time.Duration, data []byte) ([]byte, int, error) {
	log.Debug("[HTTPClientActivity:post] request url = ", url)
	log.Debug("[HTTPClientActivity:post] request header = ", header)
	log.Debug("[HTTPClientActivity:post] request body as byte = ", data)
	log.Debug("[HTTPClientActivity:post] request body as string = ", string(data))
	log.Debug("[HTTPClientActivity:post] request timeout = ", timeout.Milliseconds())
	defer log.Debug("[HTTPClientActivity:post] exit ... ")

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	//req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Error("[HTTPClientActivity:post] Error reading request. ", err)
		return nil, 500, err
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: timeout}

	log.Debug("[HTTPClientActivity:post] request header = ", req.Header)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		log.Error("[HTTPClientActivity:post] Error reading response. ", err)
		return nil, 500, err
	}
	defer resp.Body.Close()

	log.Debug("[HTTPClientActivity:post] response Status:", resp.Status)
	log.Debug("[HTTPClientActivity:post] response Headers:", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("[HTTPClientActivity:post] Error reading body. ", err)
		return nil, 500, err
	}
	log.Debug("[HTTPClientActivity:post] response body = ", string(body))

	return body, resp.StatusCode, nil
}

func (a *HTTPClientActivity) put(url string, header map[string]string, timeout time.Duration, data []byte) ([]byte, int, error) {
	log.Debug("[HTTPClientActivity:put] request url = ", url)
	log.Debug("[HTTPClientActivity:put] request header = ", header)
	log.Debug("[HTTPClientActivity:put] request body as byte = ", data)
	log.Debug("[HTTPClientActivity:put] request body as string = ", string(data))
	log.Debug("[HTTPClientActivity:put] request timeout = ", timeout.Milliseconds())
	defer log.Debug("[HTTPClientActivity:post] exit ... ")

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		log.Error("[HTTPClientActivity:put] Error reading request. ", err)
		return nil, 500, err
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: timeout}

	log.Debug("[HTTPClientActivity:put] request header = ", req.Header)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		log.Error("[HTTPClientActivity:put] Error reading response. ", err)
		return nil, 500, err
	}
	defer resp.Body.Close()

	log.Debug("[HTTPClientActivity:put] response Status:", resp.Status)
	log.Debug("[HTTPClientActivity:put] response Headers:", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("[HTTPClientActivity:put] Error reading body. ", err)
		return nil, 500, err
	}

	log.Debug("[HTTPClientActivity:put] response body = ", string(body))

	return body, resp.StatusCode, nil
}

func (a *HTTPClientActivity) getURLMapper(ctx activity.Context) (map[string]string, error) {
	myId := util.ActivityId(ctx)
	mapper := a.urlMappers[myId]

	if nil == mapper {
		a.mux.Lock()
		defer a.mux.Unlock()
		mapper = a.urlMappers[myId]
		if nil == mapper {
			mapper = make(map[string]string)
			urlsMapping, _ := ctx.GetSetting(sUrlMapping)
			log.Debug("[HTTPClientActivity:getURLMapper] Processing handlers : urlsMapping = ", urlsMapping)
			if nil != urlsMapping {
				for _, urlMapping := range urlsMapping.([]interface{}) {
					urlMappingInfo := urlMapping.(map[string]interface{})
					mapper[urlMappingInfo["Alias"].(string)] = urlMappingInfo["URL"].(string)
				}
			}
			a.urlMappers[myId] = mapper
		}
	}
	log.Debug("[HTTPClientActivity:getURLMapper] mapper = ", mapper)
	return mapper, nil
}

func (a *HTTPClientActivity) getVariableMapper(ctx activity.Context) (*kwr.KeywordMapper, map[string]string, error) {
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
			log.Debug("[HTTPClientActivity:getVariableMapper] variablesDef = ", variablesDef)
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

func (a *HTTPClientActivity) getHeader(ctx activity.Context) (map[string]string, error) {
	myId := util.ActivityId(ctx)
	header := a.header[myId]

	if nil == header {
		a.mux.Lock()
		defer a.mux.Unlock()
		header = a.header[myId]
		if nil == header {
			log.Debug("[HTTPClientActivity:getHeader] ractivity.Context = ", ctx)
			header = make(map[string]string)
			headersDef, _ := ctx.GetSetting(sHttpHeaders)
			log.Debug("[HTTPClientActivity:getheader] headersDef = ", headersDef)
			if nil != headersDef {
				for _, headerDef := range headersDef.([]interface{}) {
					headerInfo := headerDef.(map[string]interface{})
					header[headerInfo["Key"].(string)] = headerInfo["Value"].(string)
				}
			}
			log.Debug("[HTTPClientActivity:getheader] header = ", header)

			a.header[myId] = header
		}
	}
	return header, nil
}
