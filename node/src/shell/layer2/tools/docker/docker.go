package docker

import (
	"fmt"
	"kasper/src/abstract"
	modulelogger "kasper/src/core/module/logger"
	"kasper/src/shell/layer1/adapters"
	"kasper/src/shell/utils/crypto"
	"log"

	"archive/tar"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	network "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	natting "github.com/docker/go-connections/nat"
)

type Docker struct {
	app         abstract.ICore
	logger      *modulelogger.Logger
	storageRoot string
	storage     adapters.IStorage
	client      *client.Client
}

func (wm *Docker) SaRContainer(containerName string) error {
	ctx := context.Background()

	if err := wm.client.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		log.Println("Unable to stop container ", containerName, err.Error())
	}

	removeOptions := container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := wm.client.ContainerRemove(ctx, containerName, removeOptions); err != nil {
		log.Println("Unable to remove container: ", err.Error())
		return err
	}

	return nil
}

func (wm *Docker) RunContainer(imageName string, inputFile map[string]string) (string, error) {
	ctx := context.Background()

	port := "9000"
	newport, err := natting.NewPort("tcp", port)
	if err != nil {
		log.Println("Unable to create docker port")
		return "", err
	}

	hostConfig := &container.HostConfig{
		PortBindings: natting.PortMap{
			newport: []natting.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: port,
				},
			},
		},
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
		LogConfig: container.LogConfig{
			Type:   "json-file",
			Config: map[string]string{},
		},
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}
	gatewayConfig := &network.EndpointSettings{
		Gateway: "gatewayname",
	}
	networkConfig.EndpointsConfig["bridge"] = gatewayConfig

	exposedPorts := map[natting.Port]struct{}{
		newport: {},
	}

	config := &container.Config{
		Image:        imageName,
		Env:          []string{},
		ExposedPorts: exposedPorts,
		Hostname:     fmt.Sprintf("%s-hostnameexample", imageName),
	}

	cont, err := wm.client.ContainerCreate(
		context.Background(),
		config,
		hostConfig,
		networkConfig,
		nil,
		crypto.SecureUniqueString(),
	)

	if err != nil {
		log.Println(err)
		return "", err
	}

	for k, v := range inputFile {
		tarStream, err := os.Open(k)
		if err != nil {
			log.Println(err)
			return "", err
		}
		err = wm.client.CopyToContainer(ctx, cont.ID, "/app/input/"+v, tarStream, container.CopyToContainerOptions{})
		if err != nil {
			log.Println(err)
			return "", err
		}
	}

	wm.client.ContainerStart(ctx, cont.ID, container.StartOptions{})
	log.Println("Container ", cont.ID, " is created")

	return cont.ID, nil
}

func (wm *Docker) BuildImage(dockerfile string, imageName string) error {
	ctx := context.Background()

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	dockerFileReader, err := os.Open(dockerfile)
	if err != nil {
		return err
	}

	readDockerFile, err := ioutil.ReadAll(dockerFileReader)
	if err != nil {
		return err
	}

	tarHeader := &tar.Header{
		Name: dockerfile,
		Size: int64(len(readDockerFile)),
	}

	err = tw.WriteHeader(tarHeader)
	if err != nil {
		return err
	}

	_, err = tw.Write(readDockerFile)
	if err != nil {
		return err
	}

	dockerFileTarReader := bytes.NewReader(buf.Bytes())

	buildOptions := types.ImageBuildOptions{
		Context:    dockerFileTarReader,
		Dockerfile: dockerfile,
		Remove:     true,
		Tags:       []string{},
	}

	imageBuildResponse, err := wm.client.ImageBuild(
		ctx,
		dockerFileTarReader,
		buildOptions,
	)

	if err != nil {
		return err
	}

	defer imageBuildResponse.Body.Close()
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	if err != nil {
		return err
	}

	return nil
}

func NewDocker(core abstract.ICore, logger *modulelogger.Logger, storageRoot string, storage adapters.IStorage) *Docker {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Println("Unable to create docker client: ", err.Error())
	}
	wm := &Docker{
		app:         core,
		logger:      logger,
		storageRoot: storageRoot,
		storage:     storage,
		client:      client,
	}
	return wm
}
