package docker

import (
	models "kasper/src/shell/api/model"
)

type IDocker interface {
	SaRContainer(containerName string) error
	RunContainer(machineId string, pointId string, imageName string, inputFile map[string]string) (*models.File, error)
	BuildImage(dockerfile string, machineId string, imageName string) error
}
