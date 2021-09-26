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

	kwr "github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/keywordreplace"
	"github.com/SteveNY-Tibco/labs-lightcrane-contrib/common/util"
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/logger"
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

	log.Info("[HTTPClientActivity:Eval] entering ........ ")

	skipCondition := context.GetInput(iSkipCondition).(bool)
	if skipCondition {
		log.Info("[HTTPClientActivity:Eval] Skip taks : ", skipCondition)
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

	log.Info("[HTTPClientActivity:Eval] url : ", url)

	var success bool
	var errorCode int
	var data string
	if "" != url {
		method, exist := context.GetSetting(sMethod)
		if !exist {
			return false, errors.New("Query method not defined!")
		}
		log.Info("[HTTPClientActivity:Eval] Query method : ", method)

		timeout := time.Millisecond * time.Duration(10000)
		t, exist := context.GetSetting(sTimeout)
		if exist {
			timeout = time.Millisecond * time.Duration(t.(int))
		}

		header, _ := a.getHeader(context)
		var body []byte
		if "GET" == method.(string) {
			body, err = a.get(url, header, timeout)
		} else if "DELETE" == method.(string) {
			body, err = a.delete(url, header, timeout)
		} else if "POST" == method.(string) {
			reqBody, ok := context.GetInput(iBody).(string)
			if !ok {
				return false, errors.New("Invalid request body ... ")
			}
			body, err = a.post(url, header, timeout, []byte(reqBody))
		} else if "PUT" == method.(string) {
			reqBody, ok := context.GetInput(iBody).(string)
			if !ok {
				return false, errors.New("Invalid request body ... ")
			}
			body, err = a.put(url, header, timeout, []byte(reqBody))
		} else {
			return false, errors.New("Query method not support!")
		}
		if nil != err {
			log.Info("[HTTPClientActivity:Eval] Error : ", err.Error())
			success = false
			data = fmt.Sprintf("{\"Error\" : %s}", err.Error())
			errorCode = 300
		} else {
			success = true
			data = string(body)
			errorCode = 100
		}
	} else {
		log.Info("[HTTPClientActivity:Eval] Error : No URL defined!")
		success = false
		data = "{\"Error\" : \"No URL defined!\"}"
		errorCode = 300
	}

	context.SetOutput(oSuccess, success)
	context.SetOutput(oData, data)
	context.SetOutput(oErrorCode, errorCode)

	log.Info("[HTTPClientActivity:Eval] Exit ........ ")

	return true, nil
}

func (a *HTTPClientActivity) get(url string, header map[string]string, timeout time.Duration) ([]byte, error) {
	log.Info("[HTTPClientActivity:get] request url = ", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error("[HTTPClientActivity:get] Error reading request. ", err)
		return nil, err
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Error("[HTTPClientActivity:get] Error reading response. ", err)
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("[HTTPClientActivity:get] Error reading body. ", err)
		return nil, err
	}

	log.Info("[HTTPClientActivity:get] response body = ", string(body))

	return body, nil
}

func (a *HTTPClientActivity) delete(url string, header map[string]string, timeout time.Duration) ([]byte, error) {
	log.Info("[HTTPClientActivity:delete] request url = ", url)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Error("[HTTPClientActivity:get] Error reading request. ", err)
		return nil, err
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Error("[HTTPClientActivity:delete] Error reading response. ", err)
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("[HTTPClientActivity:delete] Error reading body. ", err)
		return nil, err
	}

	log.Info("[HTTPClientActivity:delete] response body = ", string(body))

	return body, nil
}

func (a *HTTPClientActivity) post(url string, header map[string]string, timeout time.Duration, data []byte) ([]byte, error) {
	log.Info("[HTTPClientActivity:post] request url = ", url)
	log.Debug("[HTTPClientActivity:post] request body = ", string(data))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Error("[HTTPClientActivity:post] Error reading request. ", err)
		return nil, err
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	//cookie := http.Cookie{Name: "cookie_name", Value: "cookie_value"}
	//req.AddCookie(&cookie)

	client := &http.Client{Timeout: timeout}

	//fmt.Println(req.Cookies())
	fmt.Println(req.Header)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		log.Error("[HTTPClientActivity:post] Error reading response. ", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Info("[HTTPClientActivity:post] response Status:", resp.Status)
	log.Info("[HTTPClientActivity:post] response Headers:", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("[HTTPClientActivity:post] Error reading body. ", err)
		return nil, err
	}

	log.Info("[HTTPClientActivity:post] response body = ", string(body))

	return body, nil
}

func (a *HTTPClientActivity) put(url string, header map[string]string, timeout time.Duration, data []byte) ([]byte, error) {
	log.Info("[HTTPClientActivity:put] request url = ", url)
	log.Debug("[HTTPClientActivity:put] request body = ", string(data))
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		log.Error("[HTTPClientActivity:post] Error reading request. ", err)
		return nil, err
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	//cookie := http.Cookie{Name: "cookie_name", Value: "cookie_value"}
	//req.AddCookie(&cookie)

	client := &http.Client{Timeout: timeout}

	//fmt.Println(req.Cookies())
	fmt.Println(req.Header)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		log.Error("[HTTPClientActivity:put] Error reading response. ", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Info("[HTTPClientActivity:put] response Status:", resp.Status)
	log.Info("[HTTPClientActivity:put] response Headers:", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("[HTTPClientActivity:put] Error reading body. ", err)
		return nil, err
	}

	log.Info("[HTTPClientActivity:put] response body = ", string(body))

	return body, nil
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
			log.Info("[HTTPClientActivity:getURLMapper] Processing handlers : urlsMapping = ", urlsMapping)
			if nil != urlsMapping {
				for _, urlMapping := range urlsMapping.([]interface{}) {
					urlMappingInfo := urlMapping.(map[string]interface{})
					mapper[urlMappingInfo["Alias"].(string)] = urlMappingInfo["URL"].(string)
				}
			}
			a.urlMappers[myId] = mapper
		}
	}
	log.Info("[HTTPClientActivity:getURLMapper] mapper = ", mapper)
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
			log.Info("[HTTPClientActivity:getVariableMapper] variablesDef = ", variablesDef)
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
			fmt.Println(ctx)
			header = make(map[string]string)
			headersDef, _ := ctx.GetSetting(sHttpHeaders)
			log.Info("[HTTPClientActivity:getheader] headersDef = ", headersDef)
			if nil != headersDef {
				for _, headerDef := range headersDef.([]interface{}) {
					headerInfo := headerDef.(map[string]interface{})
					header[headerInfo["Key"].(string)] = headerInfo["Value"].(string)
				}
			}
			log.Info("[HTTPClientActivity:getheader] header = ", header)

			a.header[myId] = header
		}
	}
	return header, nil
}
