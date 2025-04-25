package actions_plugin

import (
	"encoding/base64"
	"errors"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/state"
	inputs_machiner "kasper/src/shell/machiner/inputs/plugin"
	"kasper/src/shell/machiner/model"
	outputs_machiner "kasper/src/shell/machiner/outputs/plugin"
	"log"

	models "kasper/src/shell/api/model"
)

const pluginsTemplateName = "/machines/"

type Actions struct {
	app core.ICore
}

func Install(a *Actions) error {
	return nil
}

// Create /machines/create check [ true false false ] access [ true false false false POST ]
func (a *Actions) Create(state state.IState, input inputs_machiner.CreateInput) (any, error) {
	var (
		user    models.User
		session models.Session
	)
	trx := state.Trx()
	user = models.User{Id: a.app.Tools().Storage().GenId(input.Origin()), Typ: "machine", PublicKey: input.PublicKey, Username: input.Username + "@" + state.Dummy()}
	session = models.Session{Id: a.app.Tools().Storage().GenId(input.Origin()), UserId: user.Id}
	vm := model.Vm{MachineId: user.Id, OwnerId: state.Info().UserId()}
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
	if vm.OwnerId != state.Info().UserId() {
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
		dockerfileFolderPath := a.app.Tools().Storage().StorageRoot() + pluginsTemplateName + vm.MachineId + "/" + in
		err2 := a.app.Tools().File().SaveDataToGlobalStorage(dockerfileFolderPath, data, "Dockerfile", true)
		if err2 != nil {
			return nil, err2
		}
		err3 := a.app.Tools().Docker().BuildImage(dockerfileFolderPath+"/Dockerfile", vm.MachineId, in)
		if err3 != nil {
			log.Println(err3)
			return nil, err3
		}
	} else {
		err2 := a.app.Tools().File().SaveDataToGlobalStorage(a.app.Tools().Storage().StorageRoot()+pluginsTemplateName+vm.MachineId+"/", data, "module", true)
		if err2 != nil {
			return nil, err2
		}
		vm.Runtime = input.Runtime
		vm.Push(trx)
		if vm.Runtime == "wasm" {
			a.app.Tools().Wasm().Assign(vm.MachineId)
		} else if vm.Runtime == "elpis" {
			a.app.Tools().Elpis().Assign(vm.MachineId)
		}
	}
	return outputs_machiner.PlugInput{}, nil
}
