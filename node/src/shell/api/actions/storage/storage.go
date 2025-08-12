package actions_user

import (
	"encoding/base64"
	"errors"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/state"
	inputs_storage "kasper/src/shell/api/inputs/storage"
	models "kasper/src/shell/api/model"
	"log"
)

type Actions struct {
	App core.ICore
}

func Install(a *Actions) error {
	return nil
}

// Upload /storage/upload check [ true true true ] access [ true false false false POST ]
func (a *Actions) Upload(state state.IState, input inputs_storage.UploadDataInput) (any, error) {
	trx := state.Trx()
	if input.FileId != "" {
		if !trx.HasObj("File", input.FileId) {
			return nil, errors.New("file not found")
		}
		var file = models.File{Id: input.FileId}.Pull(trx)
		if file.OwnerId != state.Info().UserId() {
			return nil, errors.New("access to file control denied")
		}
		data, err := base64.StdEncoding.DecodeString(input.Data)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		if err := a.App.Tools().File().SaveDataToStorage(a.App.Tools().Storage().StorageRoot(), data, state.Info().PointId(), input.FileId); err != nil {
			log.Println(err)
			return nil, err
		}
		return map[string]any{}, nil
	} else {
		var file = models.File{Id: a.App.Tools().Storage().GenId(trx, input.Origin()), OwnerId: state.Info().UserId(), PointId: state.Info().PointId()}
		data, err := base64.StdEncoding.DecodeString(input.Data)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		if err := a.App.Tools().File().SaveDataToStorage(a.App.Tools().Storage().StorageRoot(), data, state.Info().PointId(), file.Id); err != nil {
			log.Println(err)
			return nil, err
		}
		file.Push(trx)
		return map[string]any{"file": file}, nil
	}
}

// UploadUserEntity /storage/uploadUserEntity check [ true false true ] access [ true false false false POST ]
func (a *Actions) UploadUserEntity(state state.IState, input inputs_storage.UploadUserEntityInput) (any, error) {
	data, err := base64.StdEncoding.DecodeString(input.Data)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if err := a.App.Tools().File().SaveDataToGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/users/"+state.Info().UserId(), data, input.EntityId, true); err != nil {
		log.Println(err)
		return nil, err
	}
	return map[string]any{}, nil
}

// UploadPointEntity /storage/uploadPointEntity check [ true false true ] access [ true false false false POST ]
func (a *Actions) UploadPointEntity(state state.IState, input inputs_storage.UploadPointEntityInput) (any, error) {
	data, err := base64.StdEncoding.DecodeString(input.Data)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if err := a.App.Tools().File().SaveDataToGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/points/"+state.Info().PointId(), data, input.EntityId, true); err != nil {
		log.Println(err)
		return nil, err
	}
	return map[string]any{}, nil
}

// DownloadUserEntity /storage/downloadUserEntity check [ true false true ] access [ true false false false POST ]
func (a *Actions) DownloadUserEntity(state state.IState, input inputs_storage.DownloadUserEntityInput) (any, error) {
	data, err := a.App.Tools().File().ReadFileFromGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/users/"+input.UserId, input.EntityId)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return map[string]any{"data": data}, nil
}

// DownloadPointEntity /storage/downloadPointEntity check [ true false true ] access [ true false false false POST ]
func (a *Actions) DownloadPointEntity(state state.IState, input inputs_storage.DownloadPointEntityInput) (any, error) {
	data, err := a.App.Tools().File().ReadFileFromGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/points/"+input.PointId, input.EntityId)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return map[string]any{"data": data}, nil
}

// Download /storage/download check [ true true true ] access [ true false false false POST ]
func (a *Actions) Download(state state.IState, input inputs_storage.DownloadInput) (any, error) {
	trx := state.Trx()
	if !trx.HasObj("File", input.FileId) {
		return nil, errors.New("file not found")
	}
	var file = models.File{Id: input.FileId}.Pull(trx)
	if file.PointId != state.Info().PointId() {
		return nil, errors.New("access to file denied")
	}
	data, err := a.App.Tools().File().ReadFileFromStorage(a.App.Tools().Storage().StorageRoot(), state.Info().PointId(), file.Id)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return map[string]any{"data": data}, nil
}
