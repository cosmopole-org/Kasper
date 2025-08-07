package actions_machine

import (
	"encoding/base64"
	"errors"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/state"
	inputs_machiner "kasper/src/shell/api/inputs/machine"
	"kasper/src/shell/api/model"
	outputs_machiner "kasper/src/shell/api/outputs/plugin"
	"kasper/src/shell/utils/future"
	"log"
)

const pluginsTemplateName = "/machines/"

type Actions struct {
	App core.ICore
}

func Install(a *Actions) error {
	a.App.ModifyState(true, func(trx trx.ITrx) error {
		vms, err := model.Vm{}.All(trx, -1, -1)
		if err != nil {
			panic(err)
		}
		for _, vm := range vms {
			if vm.Runtime == "wasm" {
				a.App.Tools().Wasm().Assign(vm.MachineId)
			} else if vm.Runtime == "elpis" {
				a.App.Tools().Elpis().Assign(vm.MachineId)
			}
		}
		return nil
	})
	return nil
}

// CreateApp /apps/create check [ true false false ] access [ true false false false POST ]
func (a *Actions) CreateApp(state state.IState, input inputs_machiner.CreateAppInput) (any, error) {
	trx := state.Trx()
	app := model.App{Id: a.App.Tools().Storage().GenId(trx, input.Origin()), OwnerId: state.Info().UserId(), ChainId: input.ChainId}
	app.Push(trx)
	a.App.Tools().Network().Chain().NotifyNewMachineCreated(input.ChainId, app.Id)
	return map[string]any{"app": app}, nil
}

// CreateMachine /machines/create check [ true false false ] access [ true false false false POST ]
func (a *Actions) CreateMachine(state state.IState, input inputs_machiner.CreateFuncInput) (any, error) {
	var (
		user    model.User
		session model.Session
	)
	trx := state.Trx()
	if !trx.HasObj("App", input.AppId) {
		return nil, errors.New("app not found")
	}
	app := model.App{Id: input.AppId}.Pull(trx)
	if app.OwnerId != state.Info().UserId() {
		return nil, errors.New("you are not owner of app")
	}
	user = model.User{Id: a.App.Tools().Storage().GenId(trx, input.Origin()), Balance: 1000, Typ: "machine", PublicKey: input.PublicKey, Username: input.Username + "@" + state.Source()}
	session = model.Session{Id: a.App.Tools().Storage().GenId(trx, input.Origin()), UserId: user.Id}
	vm := model.Vm{MachineId: user.Id, AppId: app.Id, Path: input.Path}
	user.Push(trx)
	session.Push(trx)
	vm.Push(trx)
	return outputs_machiner.CreateOutput{User: user}, nil
}

// Deploy /machines/deploy check [ true false false ] access [ true false false false POST ]
func (a *Actions) Deploy(state state.IState, input inputs_machiner.DeployInput) (any, error) {
	trx := state.Trx()
	if !trx.HasObj("Vm", input.MachineId) {
		return nil, errors.New("vm not found")
	}
	vm := model.Vm{MachineId: input.MachineId}.Pull(trx)
	if !trx.HasObj("App", vm.AppId) {
		return nil, errors.New("app not found")
	}
	app := model.App{Id: vm.AppId}.Pull(trx)
	if app.OwnerId != state.Info().UserId() {
		return nil, errors.New("access to vm denied")
	}
	data, err := base64.StdEncoding.DecodeString(input.ByteCode)
	if err != nil {
		return nil, err
	}
	if input.Runtime == "docker" {
		if input.Metadata == nil {
			return nil, errors.New("image name not provided")
		}
		imageName, ok := input.Metadata["imageName"]
		if !ok {
			return nil, errors.New("image name not provided")
		}
		in, ok2 := imageName.(string)
		if !ok2 {
			return nil, errors.New("image name is not string")
		}
		filesRaw, ok := input.Metadata["files"]
		if !ok {
			return nil, errors.New("files not provided")
		}
		files, ok2 := filesRaw.(map[string]any)
		if !ok2 {
			return nil, errors.New("files is not map")
		}
		dockerfileFolderPath := a.App.Tools().Storage().StorageRoot() + pluginsTemplateName + vm.MachineId + "/" + in
		err2 := a.App.Tools().File().SaveDataToGlobalStorage(dockerfileFolderPath, data, "Dockerfile", true)
		if err2 != nil {
			return nil, err2
		}
		for k, v := range files {
			dataStr, ok := v.(string)
			if !ok {
				err := errors.New("file bytecode not string")
				log.Println(err)
				return nil, err
			}
			data, err := base64.StdEncoding.DecodeString(dataStr)
			if err != nil {
				return nil, err
			}
			err2 := a.App.Tools().File().SaveDataToGlobalStorage(dockerfileFolderPath, data, k, true)
			if err2 != nil {
				return nil, err2
			}
		}
		future.Async(func() {
			err3 := a.App.Tools().Docker().BuildImage(dockerfileFolderPath, vm.MachineId, in)
			if err3 != nil {
				log.Println(err3)
			}
		}, false)
	} else {
		err2 := a.App.Tools().File().SaveDataToGlobalStorage(a.App.Tools().Storage().StorageRoot()+pluginsTemplateName+vm.MachineId+"/", data, "module", true)
		if err2 != nil {
			return nil, err2
		}
		vm.Runtime = input.Runtime
		vm.Push(trx)
		if vm.Runtime == "wasm" {
			a.App.Tools().Wasm().Assign(vm.MachineId)
		} else if vm.Runtime == "elpis" {
			a.App.Tools().Elpis().Assign(vm.MachineId)
		}
	}
	return outputs_machiner.PlugInput{}, nil
}

// ListApps /apps/list check [ true false false ] access [ true false false false GET ]
func (a *Actions) ListApps(state state.IState, input inputs_machiner.ListInput) (any, error) {
	trx := state.Trx()
	apps, err := model.App{}.All(trx, input.Offset, input.Count)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return map[string]any{"apps": apps}, nil
}

// ListMachs /machines/list check [ true false false ] access [ true false false false GET ]
func (a *Actions) ListMachs(state state.IState, input inputs_machiner.ListInput) (any, error) {
	trx := state.Trx()
	machines, err := model.User{}.All(trx, input.Offset, input.Count, map[string]string{"type": "machine"})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return map[string]any{"machines": machines}, nil
}
