/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package keywordreplace

import (
	"bytes"
	"fmt"

	"github.com/TIBCOSoftware/flogo-lib/logger"
)

var log = logger.GetLogger("labs-lc-keywordreplace")

func Replace(input string, lefttoken string, righttoken string, keyword string, replacement string) string {
	return NewKeywordMapper("", lefttoken, righttoken).Replace(input, map[string]interface{}{
		keyword: replacement,
	})
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

func (this *KeywordReplaceHandler) Replace(keyword string) string {
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

func (this *KeywordMapper) Replace(template string, keywordMap map[string]interface{}) string {
	if nil == keywordMap {
		return template
	}
	if "" == template {
		template = this.template
		if "" == template {
			return ""
		}
	}

	log.Debug("[KeywordMapper.replace] template = ", template)
	log.Debug("[KeywordMapper.replace] keywordMap = ", keywordMap)

	this.mh.setMap(keywordMap)
	this.document.Reset()
	this.keyword.Reset()

	scope := false
	boundary := false
	skeyword := ""
	svalue := ""

	this.mh.startToMap()
	for i := 0; i < len(template); i++ {
		log.Debugf("[KeywordMapper.replace] template[%d] = ", i, template[i])
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
				svalue = this.mh.Replace(skeyword)
				if "" == svalue {
					svalue = fmt.Sprintf("%s%s%s", this.slefttag, skeyword, this.srighttag)
				}
				log.Debug("value ->", svalue)
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

		log.Debug("[KeywordMapper.replace] scope = ", scope, ", boundary = ", boundary, ", this.keyword = ", this.keyword, ", document = ", this.document)
		if i == len(template)-1 {
			if true == scope {
				this.document.WriteString(this.keyword.String())
			}
		}
	}
	this.mh.endOfMapping(this.document.String())
	return this.mh.getResult()
}

func (this *KeywordMapper) isATag(i int, tag string, template string) bool {
	if len(template) >= len(tag) {
		for j := 0; j < len(tag); j++ {
			if tag[j] != template[i+j] {
				return false
			}
		}
		return true
	}
	return false
}
