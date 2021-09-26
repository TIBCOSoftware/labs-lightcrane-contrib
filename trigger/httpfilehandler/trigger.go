/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package httpfilehandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

var triggerMd = trigger.NewMetadata(&Settings{}, &HandlerSettings{}, &Output{})

func init() {
	_ = trigger.Register(&Trigger{}, &Factory{})
}

//-===============================-//
//     Define Trigger Factory
//-===============================-//

type Factory struct {
}

// Metadata implements trigger.Factory.Metadata
func (*Factory) Metadata() *trigger.Metadata {
	return triggerMd
}

// New implements trigger.Factory.New
func (*Factory) New(config *trigger.Config) (trigger.Trigger, error) {
	settings := &Settings{}
	err := metadata.MapToStruct(config.Settings, settings, true)
	if err != nil {
		return nil, err
	}

	return &Trigger{settings: settings}, nil
}

//-=========================-//
//      Define Trigger
//-=========================-//

var logger log.Logger

type Trigger struct {
	settings *Settings
	handlers []trigger.Handler
	mux      sync.Mutex

	httpFileHandlers []*HTTPFileHandler
}

// Init implements trigger.Init
func (this *Trigger) Initialize(ctx trigger.InitContext) error {

	this.handlers = ctx.GetHandlers()
	logger = ctx.Logger()
	this.httpFileHandlers = make([]*HTTPFileHandler, 0)

	return nil
}

// Start implements ext.Trigger.Start
func (this *Trigger) Start() error {
	logger.Info("Name: HTTPFileHandler, Port: ", this.settings.Port)
	logger.Info("Start HTTPFileHandler : subhandlers =  ", this.handlers)

	for _, handler := range this.handlers {
		logger.Info("handler: ", handler)

		handlerSetting := &HandlerSettings{}
		err := metadata.MapToStruct(handler.Settings(), handlerSetting, true)
		if err != nil {
			return err
		}

		if "" == handlerSetting.Path {
			return errors.New("Path not set yet!")
		}

		if "" == handlerSetting.BaseFolder {
			return errors.New("BaseFolder not set yet!")
		}

		baseFolder := filepath.Dir(handlerSetting.BaseFolder)
		_, err = os.Stat(baseFolder)
		if err != nil {
			if os.IsNotExist(err) {
				err := os.MkdirAll(baseFolder, os.ModePerm)
				if nil != err {
					logger.Error("Unable to create folder : ", err)
					return err
				}
			}
		}

		httpFileHandler := &HTTPFileHandler{
			handler: handler,
			path:    handlerSetting.Path,
			port:    this.settings.Port,
			folder:  handlerSetting.BaseFolder,
		}
		go httpFileHandler.start()
		this.httpFileHandlers = append(this.httpFileHandlers, httpFileHandler)
	}

	return nil
}

// Stop implements ext.Trigger.Stop
func (this *Trigger) Stop() error {
	logger.Debug("Stopping endpoints")
	for _, httpFileHandler := range this.httpFileHandlers {
		httpFileHandler.stop()
	}
	return nil
}

func (this *Trigger) HandleContent(handlerId int, filename string, filePath string) {
	this.mux.Lock()
	defer this.mux.Unlock()
}

type FileContentHandler interface {
	HandleContent(handlerId int, filename string, filePath string)
}

type HTTPFileHandler struct {
	handler trigger.Handler
	path    string
	port    string
	folder  string
}

func (this *HTTPFileHandler) upload(w http.ResponseWriter, r *http.Request) {
	logger.Info("(Serve) Request URL : ", r.URL)
	logger.Info("(Serve) Request Request URL Path = ", r.URL.Path)

	definedPathElements := strings.Split(this.path, "/")
	requestPathElements := strings.Split(r.URL.Path, "/")
	logger.Info("(Serve) Defined URL Path Elements = ", definedPathElements)
	logger.Info("(Serve) Request URL Path Elements = ", requestPathElements)

	if len(definedPathElements) != len(requestPathElements) {
		return
	}

	var filename string
	for index, value := range requestPathElements {
		if index == len(requestPathElements)-1 {
			filename = value
		} else {
			if value != requestPathElements[index] {
				break
			}
		}
	}

	fileFullname := this.folder + "/" + filename
	logger.Info("(Serve) File fullname = ", fileFullname)
	file, err := os.Create(fileFullname)
	if err != nil {
		logger.Error(err)
	}
	_, err = io.Copy(file, r.Body)
	if nil != err {
		logger.Error(err)
	}

	outputData := &Output{}
	outputData.Filename = filename
	outputData.FilePath = this.folder

	logger.Info("(FileContentHandler.HandleContent) - outputData : ", outputData)

	results, err := this.handler.Handle(context.Background(), outputData)

	if nil != err {
		logger.Errorf("Run action for handler [%s] failed for reason [%s] message lost", this.handler, err)
	}

	logger.Info(results)

	jsonString, err := json.Marshal(results["data"])

	if nil != err {
		http.Error(w, "Error processing content.",
			http.StatusInternalServerError)
	}

	w.Write([]byte(jsonString))

	logger.Infof("(FileContentHandler.HandleContent) - Trigger done")

}

/*
func (this *HTTPFileHandler) upload(w http.ResponseWriter, r *http.Request) {
	fmt.Println("File Upload Endpoint Hit")

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile(this.folder, "upload-*.png")
	if err != nil {
		fmt.Println(err)
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)
	// return that we have successfully uploaded our file!
	fmt.Fprintf(w, "Successfully Uploaded File\n")

	this.trigger.HandleContent(this.handlerId, handler.Filename, this.folder)
}
*/

func (this *HTTPFileHandler) start() {
	logger.Info("HTTPFileHandler starting at : path = ", this.path, ", port = ", this.port)

	http.HandleFunc("/", this.upload)
	err := http.ListenAndServe(fmt.Sprintf(":%s", this.port), nil)
	if err != nil {
		logger.Error("ListenAndServe: ", err)
	}
}

func (this *HTTPFileHandler) stop() {
	fmt.Println("HTTPFileHandler stopped!")
}

/*


<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
    <title>Document</title>
  </head>
  <body>
    <form
      enctype="multipart/form-data"
      action="http://localhost:8080/upload"
      method="post"
    >
      <input type="file" name="myFile" />
      <input type="submit" value="upload" />
    </form>
  </body>
</html>


*/
