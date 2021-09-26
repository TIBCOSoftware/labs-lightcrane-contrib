/*
 * Copyright © 2020. TIBCO Software Inc.
 * This file is subject to the license terms contained
 * in the license file that is distributed with this file.
 */
package capture

import (
	"context"
	"strconv"
	"sync"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"

	"github.com/docopt/docopt-go"

	"github.com/ghedo/go.pkt/capture"
	"github.com/ghedo/go.pkt/capture/file"
	"github.com/ghedo/go.pkt/capture/pcap"
	"github.com/ghedo/go.pkt/filter"
	"github.com/ghedo/go.pkt/layers"
)

const (
	cConnection     = "execConnection"
	cConnectionName = "name"
)

//-============================================-//
//   Entry point register Trigger & factory
//-============================================-//

var triggerMd = trigger.NewMetadata(&Settings{}, &HandlerSettings{}, &Output{})

func init() {
	_ = trigger.Register(&ExecListener{}, &Factory{})
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

	return &ExecListener{settings: settings}, nil
}

//-=========================-//
//      Define Trigger
//-=========================-//

var logger log.Logger

type ExecListener struct {
	metadata *trigger.Metadata
	config   *trigger.Config
	mux      sync.Mutex

	settings *Settings
	handlers []trigger.Handler
}

// implements trigger.Initializable.Initialize
func (this *ExecListener) Initialize(ctx trigger.InitContext) error {

	this.handlers = ctx.GetHandlers()
	logger = ctx.Logger()

	return nil
}

// implements ext.Trigger.Start
func (this *ExecListener) Start() error {

	logger.Info("Start")
	handlers := this.handlers

	logger.Info("Processing handlers")

	connection, exist := handlers[0].Settings()[cConnection]
	if !exist {
		return activity.NewError("SSE connection is not configured", "TGDB-SSE-4001", nil)
	}

	connectionInfo, _ := data.CoerceToObject(connection)
	if connectionInfo == nil {
		return activity.NewError("SSE connection not able to be parsed", "TGDB-SSE-4002", nil)
	}

	var serverId string
	connectionSettings, _ := connectionInfo["settings"].([]interface{})
	if connectionSettings != nil {
		for _, v := range connectionSettings {
			setting, err := data.CoerceToObject(v)
			if nil != err {
				continue
			}

			if nil != setting {
				if setting["name"] == cConnectionName {
					serverId = setting["value"].(string)
				}
			}

		}

	}

	return nil
}

// implements ext.Trigger.Stop
func (this *ExecListener) Stop() error {
	return nil
}

func (this *ExecListener) ProcessEvent(event map[string]interface{}) error {
	this.mux.Lock()
	defer this.mux.Unlock()
	logger.Debug("Got Exec event : ", event)
	outputData := &Output{}
	outputData.Event = event
	logger.Debug("Send Exec event out : ", outputData)

	_, err := this.handlers[0].Handle(context.Background(), outputData)
	if nil != err {
		logger.Info("Error -> ", err)
	}

	return err
}

func main() {

	usage := `Usage: dump [options] [<expression>]

Dump the traffic on the network (like tcpdump).

Options:
  -c <count>  Exit after receiving count packets.
  -i <iface>  Listen on interface.
  -r <file>   Read packets from file.
  -w <file>   Write the raw packets to file.`

	args, err := docopt.Parse(usage, nil, true, "", false)
	if err != nil {
		logger.Errorf("Invalid arguments: %s", err)
	}

	var count uint64

	if args["-c"] != nil {
		count, err = strconv.ParseUint(args["-c"].(string), 10, 64)
		if err != nil {
			logger.Errorf("Error parsing count: %s", err)
		}
	}

	var src capture.Handle

	if args["-i"] != nil {
		src, err = pcap.Open(args["-i"].(string))
		if err != nil {
			logger.Errorf("Error opening iface: %s", err)
		}
	} else if args["-r"] != nil {
		src, err = file.Open(args["-r"].(string))
		if err != nil {
			logger.Errorf("Error opening file: %s", err)
		}
	} else {
		logger.Errorf("Must select a source (either -i or -r)")
	}
	defer src.Close()

	var dst capture.Handle

	if args["-w"] != nil {
		dst, err = file.Open(args["-w"].(string))
		if err != nil {
			logger.Errorf("Error opening file: %s", err)
		}
		defer dst.Close()
	}

	err = src.Activate()
	if err != nil {
		logger.Errorf("Error activating source: %s", err)
	}

	if args["<expression>"] != nil {
		expr := args["<expression>"].(string)

		flt, err := filter.Compile(expr, src.LinkType(), false)
		if err != nil {
			logger.Errorf("Error parsing filter: %s", err)
		}
		defer flt.Cleanup()

		err = src.ApplyFilter(flt)
		if err != nil {
			logger.Errorf("Error appying filter: %s", err)
		}
	}

	var i uint64

	for {
		buf, err := src.Capture()
		if err != nil {
			logger.Errorf("Error: %s", err)
			break
		}

		if buf == nil {
			break
		}

		i++

		if dst == nil {
			rcv_pkt, err := layers.UnpackAll(buf, src.LinkType())
			if err != nil {
				logger.Errorf("Error: %s\n", err)
			}

			logger.Info(rcv_pkt)
		} else {
			dst.Inject(buf)
		}

		if count > 0 && i >= count {
			break
		}
	}
}
