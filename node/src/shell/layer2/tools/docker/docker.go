package docker

import (
	"kasper/src/abstract"
	modulelogger "kasper/src/core/module/logger"
	"kasper/src/shell/layer1/adapters"
	"kasper/src/shell/utils/crypto"
	"log"
	"strings"

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

func WriteToTar(inputFiles map[string]string) string {
	tarId := crypto.SecureUniqueString()
	buf, err := os.Create(tarId)
	if err != nil {
		log.Println(err)
		return ""
	}
	tw := tar.NewWriter(buf)
	defer func() {
		tw.Close()
		buf.Close()
	}()
	for path, name := range inputFiles {
		fr, _ := os.Open(path)
		defer fr.Close()
		fi, _ := fr.Stat()
		h := new(tar.Header)
		if fi.IsDir() {
			h.Typeflag = tar.TypeDir
		} else {
			h.Typeflag = tar.TypeReg
		}
		h.Name = name
		h.Size = fi.Size()
		h.Mode = int64(fi.Mode())
		h.ModTime = fi.ModTime()
		_ = tw.WriteHeader(h)
		if !fi.IsDir() {
			_, _ = io.Copy(tw, fr)
		}
	}
	return tarId
}

func (wm *Docker) RunContainer(machineId string, imageName string, inputFile map[string]string) (string, error) {
	ctx := context.Background()

	config := &container.Config{
		Image:        strings.Join(strings.Split(machineId, "@"), "_") + "/" + imageName,
		Env:          []string{},
	}

	cont, err := wm.client.ContainerCreate(
		ctx,
		config,
		&container.HostConfig{},
		&network.NetworkingConfig{},
		nil,
		crypto.SecureUniqueString(),
	)

	if err != nil {
		log.Println(err)
		return "", err
	}

	tarId := WriteToTar(inputFile)
	tarStream, err := os.Open(tarId)
	if err != nil {
		log.Println(err)
		return "", err
	}

	err = wm.client.CopyToContainer(ctx, cont.ID, "/app/input", tarStream, container.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	if err != nil {
		log.Println(err)
		return "", err
	}

	err = wm.client.ContainerStart(ctx, cont.ID, container.StartOptions{})
	if err != nil {
		log.Println(err)
		return "", err
	}

	log.Println("Container ", cont.ID, " is created")

	waiter, err := wm.client.ContainerAttach(ctx, cont.ID, container.AttachOptions{
		Stderr: true,
		Stdout: true,
		Stdin:  true,
		Stream: true,
	})

	go io.Copy(os.Stdout, waiter.Reader)
	go io.Copy(os.Stderr, waiter.Reader)

	if err != nil {
		log.Println(err)
		return "", err
	}

	statusCh, errCh := wm.client.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Println(err)
			return "", err
		}
	case <-statusCh:
	}

	return cont.ID, nil
}

func (wm *Docker) BuildImage(dockerfile string, machineId string, imageName string) error {
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
		Tags:       []string{strings.Join(strings.Split(machineId, "@"), "_") + "/" + imageName},
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
