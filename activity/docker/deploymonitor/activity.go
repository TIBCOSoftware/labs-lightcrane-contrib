package deploymonitor

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/metadata"
)

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

func init() {
	_ = activity.Register(&Activity{}, New)
}

func New(ctx activity.InitContext) (activity.Activity, error) {
	settings := &Settings{}
	err := metadata.MapToStruct(ctx.Settings(), settings, true)
	if err != nil {
		return nil, err
	}

	act := &Activity{
		settings: settings,
	}
	return act, nil
}

type Activity struct {
	settings *Settings
}

func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

func (a *Activity) Eval(ctx activity.Context) (done bool, err error) {
	ctx.Logger().Info("(AirDeployMonitor:Eval) entering ........ ")
	defer ctx.Logger().Info("(AirDeployMonitor:Eval) exit ........ ")

	input := &Input{}
	err = ctx.GetInputObject(input)
	if err != nil {
		return true, err
	}

	registeredDeploymentMap := make(map[string]interface{})
	for _, registeredDeployment := range input.CurrentRegisteredDeployments {
		registeredDeploymentMap[registeredDeployment.(map[string]interface{})["ID"].(string)] = registeredDeployment
	}

	ctx.Logger().Info("(AirDeployMonitor:Eval) registeredDeploymentMap : ", registeredDeploymentMap)

	/* query docker container */
	dctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false, err
	}

	containers, err := cli.ContainerList(dctx, types.ContainerListOptions{All: true})
	if err != nil {
		return false, err
	}

	currentDeploymnts := make([]interface{}, 0)
	for _, container := range containers {
		ctx.Logger().Debug(container.Names[0] + "-" + container.Status)
		containerName := container.Names[0]
		ID := containerName[1:]

		ctx.Logger().Debug("(AirDeployMonitor:Eval) ID : ", ID)

		/*
			Delete
			ID
			Domain
			Name
			Data {}
			Status
			ContainerStatus
			Reporter
			LastModified
			ErrorCode
			ErrorMessage
		*/
		if nil != registeredDeploymentMap[ID] {
			if input.Location == registeredDeploymentMap[ID].(map[string]interface{})["Location"].(string) {
				status := registeredDeploymentMap[ID].(map[string]interface{})["Properties"].(map[string]interface{})["Status"]
				name := containerName[strings.Index(containerName[strings.Index(containerName, "_")+1:], "_")+len(containerName[0:strings.Index(containerName, "_")])+1:]
				if "Deploying" == status {
					status = "Deployed"
				}
				currentDeploymnts = append(currentDeploymnts, map[string]interface{}{
					"ID":              ID,
					"Domain":          containerName[1:strings.Index(containerName, name)],
					"Name":            name[1:],
					"Status":          status,
					"ContainerStatus": container.Status,
					"Reporter":        "Deployer",
					"LastModified":    time.Now().Unix(),
					"Delete":          false,
					"Location":        input.Location,
				})
			}
			delete(registeredDeploymentMap, ID)
		}
	}

	for ID, registeredDeployment := range registeredDeploymentMap {
		status := registeredDeployment.(map[string]interface{})["Properties"].(map[string]interface{})["Status"]
		if "Undeploying" == status {
			currentDeploymnts = append(currentDeploymnts, map[string]interface{}{
				"ID":       ID,
				"Delete":   true,
				"Location": input.Location,
			})
		}
	}

	ctx.Logger().Info("(AirDeployMonitor:Eval) currentDeploymnts : ", currentDeploymnts)

	err = ctx.SetOutput("data", currentDeploymnts)
	if err != nil {
		return false, err
	}

	return true, nil
}
