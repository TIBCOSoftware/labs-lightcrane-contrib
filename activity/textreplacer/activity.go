/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package textreplacer

import (
	"errors"
	"sync"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/TIBCOSoftware/flogo-lib/logger"

	kwr "github.com/TIBCOSoftware/labs-lightcrane-contrib/common/keywordreplace"
	"github.com/TIBCOSoftware/labs-lightcrane-contrib/common/util"
)

// activityLogger is the default logger for the Filter Activity
var log = logger.GetLogger("activity-textreplacer")

const (
	sLeftToken      = "leftToken"
	sRightToken     = "rightToken"
	iInputDocument  = "inputDocument"
	iReplacements   = "replacements"
	oOutputDocument = "outputDocument"
)

// Mapping is an Activity that is used to Filter a message to the console
type TextReplacer struct {
	metadata    *activity.Metadata
	initialized bool
	mux         sync.Mutex
	tokenMap    map[string][]string
}

// NewActivity creates a new AppActivity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	aCSVParserActivity := &TextReplacer{
		metadata: metadata,
		tokenMap: make(map[string][]string),
	}
	return aCSVParserActivity
}

// Metadata returns the activity's metadata
func (a *TextReplacer) Metadata() *activity.Metadata {
	return a.metadata
}

// Eval implements api.Activity.Eval - Filters the Message
func (a *TextReplacer) Eval(ctx activity.Context) (done bool, err error) {

	log.Debug("(TextReplacer.Eval) entering ..... ")
	defer log.Debug("(TextReplacer.Eval) exit ..... ")

	tokens, err := a.getTokens(ctx)
	if nil != err {
		return false, err
	}

	inputDocument, ok := ctx.GetInput(iInputDocument).(string)
	if !ok {
		return false, errors.New("Invalid document ... ")
	}

	replacements := ctx.GetInput(iReplacements).(*data.ComplexObject)
	replacementMap := replacements.Value.(map[string]interface{})

	mapper := kwr.NewKeywordMapper(inputDocument, tokens[0], tokens[1])
	document := mapper.Replace("", replacementMap)

	log.Debug("document = ", document)

	ctx.SetOutput(oOutputDocument, document)

	return true, nil
}

func (a *TextReplacer) getTokens(context activity.Context) ([]string, error) {
	myId := util.ActivityId(context)
	tokens := a.tokenMap[myId]
	log.Debug("tokenMap : ", a.tokenMap, ", myId : ", myId)
	if nil == tokens {
		a.mux.Lock()
		defer a.mux.Unlock()
		tokens = a.tokenMap[myId]
		if nil == tokens {
			tokens = make([]string, 2)
			leftToken, _ := context.GetSetting(sLeftToken)
			if nil != leftToken {
				tokens[0] = leftToken.(string)
			}
			rightToken, _ := context.GetSetting(sRightToken)
			if nil != rightToken {
				tokens[1] = rightToken.(string)
			}
			a.tokenMap[myId] = tokens
		}
	}

	return tokens, nil
}
