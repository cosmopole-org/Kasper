package actions_user

import (
	"encoding/base64"
	"errors"
	"fmt"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/state"
	inputs_storage "kasper/src/shell/api/inputs/storage"
	models "kasper/src/shell/api/model"
	modulemodel "kasper/src/shell/layer1/model"
	"log"
)

type Actions struct {
	app core.ICore
}

func Install(a *Actions) error {
	return nil
}

// Upload /storage/upload check [ true true true ] access [ true false false false POST ]
func (a *Actions) Upload(state state.IState, input inputs_storage.UploadInput) (any, error) {
	trx := state.Trx()
	if input.FileId != "" {
		if !trx.HasObj("File", input.FileId) {
			return nil, errors.New("file not found")	
		}
		var file = models.File{Id: input.FileId}.Pull(trx)
		if file.OwnerId != state.Info().UserId() {
			return nil, errors.New("access to file control denied")
		}
		if err := a.app.Tools().File().SaveFileToStorage(a.app.Tools().Storage().StorageRoot(), input.Data, state.Info().PointId(), input.FileId); err != nil {
			log.Println(err)
			return nil, err
		}
		return map[string]any{}, nil
	} else {
		var file = models.File{Id: a.app.Tools().Storage().GenId(input.Origin()), OwnerId: state.Info().UserId(), PointId: state.Info().PointId()}
		if err := a.app.Tools().File().SaveFileToStorage(a.app.Tools().Storage().StorageRoot(), input.Data, state.Info().PointId(), file.Id); err != nil {
			log.Println(err)
			return nil, err
		}
		file.Push(trx)
		return map[string]any{"file": file}, nil
	}
}

// UploadData /storage/uploadData check [ true true true ] access [ true false false false POST ]
func (a *Actions) UploadData(state state.IState, input inputs_storage.UploadDataInput) (any, error) {
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
		if err := a.app.Tools().File().SaveDataToStorage(a.app.Tools().Storage().StorageRoot(), data, state.Info().PointId(), input.FileId); err != nil {
			log.Println(err)
			return nil, err
		}
		return map[string]any{}, nil
	} else {
		var file = models.File{Id: a.app.Tools().Storage().GenId(input.Origin()), OwnerId: state.Info().UserId(), PointId: state.Info().PointId()}
		data, err := base64.StdEncoding.DecodeString(input.Data)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		if err := a.app.Tools().File().SaveDataToStorage(a.app.Tools().Storage().StorageRoot(), data, state.Info().PointId(), file.Id); err != nil {
			log.Println(err)
			return nil, err
		}
		file.Push(trx)
		return map[string]any{"file": file}, nil
	}
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
	return modulemodel.Command{Value: "sendFile", Data: fmt.Sprintf("%s/files/%s/%s", a.app.Tools().Storage().StorageRoot(), state.Info().PointId(), input.FileId)}, nil
}
