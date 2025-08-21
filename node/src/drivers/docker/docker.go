package docker

import (
	"errors"
	"kasper/src/abstract/adapters/docker"
	"kasper/src/abstract/adapters/file"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/trx"
	models "kasper/src/shell/api/model"
	"kasper/src/shell/utils/crypto"
	"kasper/src/shell/utils/future"

	"archive/tar"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	network "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Docker struct {
	app         core.ICore
	storageRoot string
	storage     storage.IStorage
	file        file.IFile
	client      *client.Client
}

func (wm *Docker) SaRContainer(machineId string, imageName string, containerName string) error {
	cn := strings.Join(strings.Split(machineId, "@"), "_") + "_" + imageName + "_" + containerName
	ctx := context.Background()

	if err := wm.client.ContainerStop(ctx, cn, container.StopOptions{}); err != nil {
		log.Println("Unable to stop container ", cn, err.Error())
	}

	removeOptions := container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := wm.client.ContainerRemove(ctx, cn, removeOptions); err != nil {
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

func WriteToTarDirectly(inputFiles map[string][]byte) string {
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
	for name, content := range inputFiles {
		h := new(tar.Header)
		h.Typeflag = tar.TypeReg
		h.Name = name
		h.Size = int64(len(content))
		h.Mode = int64(0600)
		_ = tw.WriteHeader(h)
		_, _ = tw.Write(content)
	}
	return tarId
}

func (wm *Docker) readFromTar(tr *tar.Reader, machineId string, pointId string) (*models.File, error) {
	header, err := tr.Next()

	switch {
	case err == io.EOF:
		return nil, err
	case err != nil:
		return nil, err
	}

	if header.Typeflag == tar.TypeReg {
		var file *models.File
		wm.app.ModifyState(false, func(trx trx.ITrx) error {
			file = &models.File{Id: wm.storage.GenId(trx, wm.app.Id()), OwnerId: machineId, PointId: pointId}
			return nil
		})
		if err := wm.file.SaveTarFileItemToStorage(wm.storageRoot, tr, pointId, file.Id); err != nil {
			log.Println(err)
			return nil, err
		}
		wm.app.ModifyState(false, func(trx trx.ITrx) error {
			file.Push(trx)
			return nil
		})
		return file, nil
	}
	err2 := errors.New("not a file")
	log.Println(err2)
	return nil, err2
}

func (wm *Docker) CopyToContainer(machineId string, imageName string, containerName string, fileName string, content string) error {
	cn := strings.Join(strings.Split(machineId, "@"), "_") + "_" + imageName + "_" + containerName
	tarId := WriteToTarDirectly(map[string][]byte{fileName: []byte(content)})
	tarStream, err := os.Open(tarId)
	if err != nil {
		log.Println(err)
		return err
	}
	ctx := context.Background()
	err = wm.client.CopyToContainer(ctx, cn, "/app/input", tarStream, container.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (wm *Docker) RunContainer(machineId string, pointId string, imageName string, containerName string, inputFile map[string]string) (*models.File, error) {

	cn := strings.Join(strings.Split(machineId, "@"), "_") + "_" + imageName + "_" + containerName

	ctx := context.Background()

	config := &container.Config{
		Image: strings.Join(strings.Split(machineId, "@"), "_") + "/" + imageName,
		Env:   []string{},
	}

	_, err := wm.client.ContainerCreate(
		ctx,
		config,
		&container.HostConfig{
			LogConfig: container.LogConfig{
				Type:   "json-file",
				Config: map[string]string{},
			},
			Runtime: "runsc",
		},
		&network.NetworkingConfig{},
		nil,
		cn,
	)

	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer wm.SaRContainer(machineId, imageName, containerName)
	future.Async(func() {
		time.Sleep(60 * time.Minute)
		// wm.SaRContainer(cn)
	}, false)

	tarId := WriteToTar(inputFile)
	tarStream, err := os.Open(tarId)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = wm.client.CopyToContainer(ctx, cn, "/app/input", tarStream, container.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = wm.client.ContainerStart(ctx, cn, container.StartOptions{})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Println("Container ", cn, " is created")

	waiter, err := wm.client.ContainerAttach(ctx, cn, container.AttachOptions{
		Stderr: true,
		Stdout: true,
		Stdin:  true,
		Stream: true,
	})

	go io.Copy(os.Stdout, waiter.Reader)
	go io.Copy(os.Stderr, waiter.Reader)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	statusCh, errCh := wm.client.ContainerWait(ctx, cn, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Println(err)
			return nil, err
		}
	case <-statusCh:
	}

	reader, _, err := wm.client.CopyFromContainer(ctx, cn, "/app/output")
	if err != nil {
		log.Println(err)
		return nil, nil
	}
	defer reader.Close()
	r := tar.NewReader(reader)
	file, err := wm.readFromTar(r, machineId, pointId)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return file, nil
}

func (wm *Docker) BuildImage(dockerfile string, machineId string, imageName string, outputChan chan string) error {
	ctx := context.Background()

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	dockerFileReader, err := os.Open(dockerfile + "/Dockerfile")
	if err != nil {
		return err
	}
	defer dockerFileReader.Close()
	readDockerFile, err := ioutil.ReadAll(dockerFileReader)
	if err != nil {
		return err
	}
	tarHeader := &tar.Header{
		Name: dockerfile + "/Dockerfile",
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

	files, err := ioutil.ReadDir(dockerfile)
	if err != nil {
		log.Println(err)
		return err
	}
	for _, file := range files {
		err := func() error {
			reader, err := os.Open(dockerfile + "/" + file.Name())
			if err != nil {
				return err
			}
			defer reader.Close()
			readFile, err := ioutil.ReadAll(reader)
			if err != nil {
				return err
			}
			tarHeader := &tar.Header{
				Name: file.Name(),
				Size: int64(len(readFile)),
			}
			err = tw.WriteHeader(tarHeader)
			if err != nil {
				return err
			}
			_, err = tw.Write(readFile)
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			log.Println(err)
			return err
		}
	}

	dockerFileTarReader := bytes.NewReader(buf.Bytes())

	buildOptions := types.ImageBuildOptions{
		Context:    dockerFileTarReader,
		Dockerfile: dockerfile + "/Dockerfile",
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

	var outBuf, errBuf bytes.Buffer
	outputDone := make(chan error)

	go func() {
		_, err = stdcopy.StdCopy(&outBuf, &errBuf, imageBuildResponse.Body)
		outputChan <- outBuf.String()
		outBuf.Reset()
		outputChan <- errBuf.String()
		outBuf.Reset()
		outputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			log.Println(err)
			outputChan <- err.Error()
			outputChan <- ""
			return err
		}
		break

	case <-ctx.Done():
		outputChan <- ctx.Err().Error()
		outputChan <- ""
	}

	return nil
}

func (wm *Docker) ExecContainer(machineId string, imageName string, containerName string, command string) (string, error) {

	cn := strings.Join(strings.Split(machineId, "@"), "_") + "_" + imageName + "_" + containerName

	ctx := context.Background()

	config := container.ExecOptions{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          strings.Split(command, " "),
	}

	res, err := wm.client.ContainerExecCreate(ctx, cn, config)
	if err != nil {
		return "", err
	}
	execId := res.ID

	resp, err := wm.client.ContainerExecAttach(ctx, execId, container.ExecAttachOptions{})
	if err != nil {
		return "", err
	}
	defer resp.Close()

	var outBuf, errBuf bytes.Buffer
	outputDone := make(chan error)

	go func() {
		// StdCopy demultiplexes the stream into two buffers
		_, err = stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader)
		outputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			return "", err
		}
		break

	case <-ctx.Done():
		return "", ctx.Err()
	}

	stdout, err := ioutil.ReadAll(&outBuf)
	if err != nil {
		return "", err
	}
	stderr, err := ioutil.ReadAll(&errBuf)
	if err != nil {
		return "", err
	}

	log.Println("output of exec :", string(stdout))

	return string(stdout) + string(stderr), nil
}

func NewDocker(core core.ICore, storageRoot string, storage storage.IStorage, file file.IFile) docker.IDocker {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Println("Unable to create docker client: ", err.Error())
	}
	wm := &Docker{
		app:         core,
		storageRoot: storageRoot,
		storage:     storage,
		file:        file,
		client:      client,
	}
	return wm
}
