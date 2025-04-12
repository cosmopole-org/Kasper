package actions_user

import (
	"errors"
	"kasper/src/abstract"
	inputs_storage "kasper/src/shell/api/inputs/storage"
	models "kasper/src/shell/api/model"
	"kasper/src/shell/layer1/adapters"
	modulestate "kasper/src/shell/layer1/module/state"
	module_model "kasper/src/shell/layer2/model"
	"log"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	err := s.Db().AutoMigrate(&models.User{})
	if err != nil {
		return err
	}
	err2 := s.Db().AutoMigrate(&models.Session{})
	if err2 != nil {
		return err2
	}
	return nil
}

// Upload /storage/upload check [ true true true ] access [ true false false false POST ]
func (a *Actions) Upload(s abstract.IState, input inputs_storage.UploadInput) (any, error) {
	toolbox := abstract.UseToolbox[*module_model.ToolboxL2](a.Layer.Core().Get(2).Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()
	if input.FileId != "" {
		var file = models.File{Id: input.FileId}
		trx.Db().First(&file)
		trx.ClearError()
		if file.SenderId != state.Info().UserId() {
			return nil, errors.New("access to file control denied")
		}
		if err := toolbox.File().SaveFileToStorage(toolbox.Storage().StorageRoot(), input.Data, state.Info().TopicId(), input.FileId); err != nil {
			log.Println(err)
			return nil, err
		}
		return map[string]any{}, nil
	} else {
		var file = models.File{Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), SenderId: state.Info().UserId(), TopicId: state.Info().TopicId()}
		if err := toolbox.File().SaveFileToStorage(toolbox.Storage().StorageRoot(), input.Data, state.Info().TopicId(), file.Id); err != nil {
			log.Println(err)
			return nil, err
		}
		trx.Db().Create(&file)
		return map[string]any{"file": file}, nil
	}
}
