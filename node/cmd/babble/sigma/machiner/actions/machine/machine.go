package actions_plugin

import (
	"encoding/base64"
	"errors"
	"fmt"
	"kasper/cmd/babble/sigma/abstract"
	"kasper/cmd/babble/sigma/layer1/adapters"
	modulestate "kasper/cmd/babble/sigma/layer1/module/state"
	modulemodel "kasper/cmd/babble/sigma/layer2/model"
	inputs_machiner "kasper/cmd/babble/sigma/machiner/inputs/plugin"
	"kasper/cmd/babble/sigma/machiner/model"
	outputs_machiner "kasper/cmd/babble/sigma/machiner/outputs/plugin"
	"log"
	"strconv"

	models "kasper/cmd/babble/sigma/api/model"
	toolbox2 "kasper/cmd/babble/sigma/layer1/module/toolbox"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const pluginsTemplateName = "/machines/"

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	err := s.Db().AutoMigrate(&model.Vm{})
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func convertRowIdToCode(rowId uint) string {
	idStr := fmt.Sprintf("%d", rowId)
	for len(idStr) < 6 {
		idStr = "0" + idStr
	}
	var c = ""
	for i := 0; i < len(idStr); i++ {
		if i < 3 {
			digit, err := strconv.ParseInt(idStr[i:i+1], 10, 32)
			if err != nil {
				fmt.Println(err)
				return ""
			}
			c += string(rune('A' + digit))
		} else {
			c += idStr[i : i+1]
		}
	}
	return c
}

// Create /machines/create check [ true false false ] access [ true false false false POST ]
func (a *Actions) Create(s abstract.IState, input inputs_machiner.CreateInput) (any, error) {
	toolbox := abstract.UseToolbox[*toolbox2.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var (
		user models.User
	)
	trx := state.Trx()
	user = models.User{Metadata: datatypes.JSON([]byte(`{}`)), Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), Typ: "machine", PublicKey: input.PublicKey, Username: input.Username + "@" + a.Layer.Core().Id(), Name: "", Avatar: ""}
	err := trx.Db().Create(&user).Error
	if err != nil {
		return nil, err
	}
	trx.Db().First(&user)
	code := convertRowIdToCode(uint(user.Number))
	err2 := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&models.User{}).Where("id = ?", user.Id) }, &user, "metadata", "code", code)
	if err2 != nil {
		return nil, err2
	}
	vm := model.Vm{MachineId: user.Id, OwnerId: state.Info().UserId()}
	trx.Db().Create(&vm)
	trx.Mem().Put("code::"+code, user.Id)
	return outputs_machiner.CreateOutput{User: user}, nil
}

// Deploy /machines/deploy check [ true false false ] access [ true false false false POST ]
func (a *Actions) Deploy(s abstract.IState, input inputs_machiner.DeployInput) (any, error) {
	toolbox := abstract.UseToolbox[*modulemodel.ToolboxL2](a.Layer.Core().Get(2).Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()

	vm := model.Vm{MachineId: input.MachineId}
	e := trx.Db().First(&vm).Error
	if e != nil {
		return nil, e
	}
	if vm.OwnerId != state.Info().UserId() {
		return nil, errors.New("access to vm denied")
	}

	data, err := base64.StdEncoding.DecodeString(input.ByteCode)
	if err != nil {
		return nil, err
	}

	err2 := toolbox.File().SaveDataToGlobalStorage(toolbox.Storage().StorageRoot()+pluginsTemplateName+vm.MachineId, data, "module", true)
	if err2 != nil {
		return nil, err2
	}

	vm.Runtime = input.Runtime
	trx.Db().Save(&vm)

	if vm.Runtime == "wasm" {
		toolbox.Wasm().Assign(vm.MachineId)
	} else if vm.Runtime == "elpis" {
		toolbox.Elpis().Assign(vm.MachineId)
	}

	return outputs_machiner.PlugInput{}, nil
}
