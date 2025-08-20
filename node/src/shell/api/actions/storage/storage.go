package actions_user

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/state"
	inputs_storage "kasper/src/shell/api/inputs/storage"
	models "kasper/src/shell/api/model"
	"kasper/src/shell/utils/future"
	"log"
	"net/http"
	"strconv"
)

type Actions struct {
	App core.ICore
}

func registerRoute(path string, handler func(w http.ResponseWriter, r *http.Request)) {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			r := recover()
			if r != nil {
				var err error
				switch t := r.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = t
				default:
					err = errors.New("unknown error")
				}
				log.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			// Preflight ends here
			w.WriteHeader(http.StatusOK)
			return
		}
		handler(w, r)
	})
}

func Install(a *Actions) error {
	registerRoute("/storage/downloadUserEntity", func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("User-Id")
		inputLengthStr := r.Header.Get("Input-Length")
		ilI64, err := strconv.ParseInt(inputLengthStr, 10, 32)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		inputLength := int(ilI64)
		var input inputs_storage.DownloadUserEntityInput
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		inputBody := body[0:inputLength]
		signature := body[inputLength:]
		if success, _, _ := a.App.Tools().Security().AuthWithSignature(userId, inputBody, string(signature)); !success {
			http.Error(w, "signature verification failed", http.StatusForbidden)
			return
		}
		err = json.Unmarshal(inputBody, &input)
		if err != nil {
			log.Printf("Error parsing body: %v", err)
			http.Error(w, "can't parse body", http.StatusBadRequest)
			return
		}
		data, err := a.App.Tools().File().ReadFileFromGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/users/"+input.UserId, input.EntityId)
		if err != nil {
			log.Println(err)
			http.Error(w, "can't read file", http.StatusBadRequest)
			return
		}
		w.Write([]byte(data))
	})
	registerRoute("/storage/uploadUserEntity", func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("User-Id")
		inputLengthStr := r.Header.Get("Input-Length")
		ilI64, err := strconv.ParseInt(inputLengthStr, 10, 32)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		inputLength := int(ilI64)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		inputBody := body[0:inputLength]
		signature := body[inputLength:]
		if success, _, _ := a.App.Tools().Security().AuthWithSignature(userId, inputBody, string(signature)); !success {
			http.Error(w, "signature verification failed", http.StatusForbidden)
			return
		}
		machineId := r.Header.Get("Machine-Id")
		entityId := r.Header.Get("Entity-Id")
		var e error
		a.App.ModifyState(false, func(trx trx.ITrx) error {
			data := inputBody
			if machineId == "" {
				if err := a.App.Tools().File().SaveDataToGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/users/"+userId, data, entityId, true); err != nil {
					log.Println(err)
					e = err
					return err
				}
			} else {
				vm := models.Vm{MachineId: machineId}.Pull(trx)
				app := models.App{Id: vm.AppId}.Pull(trx)
				if app.OwnerId != userId {
					e = errors.New("you are not owner of this machine")
					return err
				}
				if err := a.App.Tools().File().SaveDataToGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/users/"+vm.MachineId, data, entityId, true); err != nil {
					log.Println(err)
					e = err
					return err
				}
			}
			return nil
		})
		if e == nil {
			w.Write([]byte("{ \"resCode\": 0, \"obj\": {} }"))
		} else {
			b, _ := json.Marshal(map[string]any{
				"resCode": 1,
				"message": e.Error(),
			})
			w.Write(b)
		}
	})
	registerRoute("/storage/uploadPointEntity", func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("User-Id")
		inputLengthStr := r.Header.Get("Input-Length")
		ilI64, err := strconv.ParseInt(inputLengthStr, 10, 32)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		inputLength := int(ilI64)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		inputBody := body[0:inputLength]
		signature := body[inputLength:]
		if success, _, _ := a.App.Tools().Security().AuthWithSignature(userId, inputBody, string(signature)); !success {
			http.Error(w, "signature verification failed", http.StatusForbidden)
			return
		}
		pointId := r.Header.Get("Point-Id")
		entityId := r.Header.Get("Entity-Id")
		var e error
		a.App.ModifyState(false, func(trx trx.ITrx) error {
			data := inputBody
			if trx.GetLink("admin::"+pointId+"::"+userId) == "" {
				e = errors.New("you are not admin")
				return err
			}
			if err := a.App.Tools().File().SaveDataToGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/points/"+pointId, data, entityId, true); err != nil {
				log.Println(err)
				e = err
				return err
			}
			future.Async(func() {
				a.App.Tools().Signaler().SignalGroup("storage/updatePointEntity", pointId, map[string]any{"pointId": pointId, "entityId": entityId}, true, []string{})
			}, false)
			return nil
		})
		if e == nil {
			w.Write([]byte("{ \"resCode\": 0, \"obj\": {} }"))
		} else {
			b, _ := json.Marshal(map[string]any{
				"resCode": 1,
				"message": e.Error(),
			})
			w.Write(b)
		}
	})
	registerRoute("/storage/downloadPointEntity", func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("User-Id")
		inputLengthStr := r.Header.Get("Input-Length")
		ilI64, err := strconv.ParseInt(inputLengthStr, 10, 32)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		inputLength := int(ilI64)
		var input inputs_storage.DownloadPointEntityInput
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		inputBody := body[0:inputLength]
		signature := body[inputLength:]
		if success, _, _ := a.App.Tools().Security().AuthWithSignature(userId, inputBody, string(signature)); !success {
			http.Error(w, "signature verification failed", http.StatusForbidden)
			return
		}
		err = json.Unmarshal(inputBody, &input)
		if err != nil {
			log.Printf("Error parsing body: %v", err)
			http.Error(w, "can't parse body", http.StatusBadRequest)
			return
		}
		data, err := a.App.Tools().File().ReadFileFromGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/points/"+input.PointId, input.EntityId)
		if err != nil {
			log.Println(err)
			http.Error(w, "can't read file", http.StatusBadRequest)
			return
		}
		w.Write([]byte(data))
	})
	future.Async(func() {
		http.ListenAndServe(":3000", nil)
	}, false)
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
	trx := state.Trx()
	data, err := base64.StdEncoding.DecodeString(input.Data)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if input.MachineId == "" {
		if err := a.App.Tools().File().SaveDataToGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/users/"+state.Info().UserId(), data, input.EntityId, true); err != nil {
			log.Println(err)
			return nil, err
		}
	} else {
		vm := models.Vm{MachineId: input.MachineId}.Pull(trx)
		app := models.App{Id: vm.AppId}.Pull(trx)
		if app.OwnerId != state.Info().UserId() {
			return nil, errors.New("you are not owner of this machine")
		}
		if err := a.App.Tools().File().SaveDataToGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/users/"+vm.MachineId, data, input.EntityId, true); err != nil {
			log.Println(err)
			return nil, err
		}
	}
	return map[string]any{}, nil
}

// DeleteUserEntity /storage/deleteUserEntity check [ true false false ] access [ true false false false POST ]
func (a *Actions) DeleteUserEntity(state state.IState, input inputs_storage.DeleteUserEntityInput) (any, error) {
	trx := state.Trx()
	if input.MachineId == "" {
		if err := a.App.Tools().File().DeleteFileFromGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/users/"+state.Info().UserId(), input.EntityId, true); err != nil {
			log.Println(err)
			return nil, err
		}
	} else {
		vm := models.Vm{MachineId: input.MachineId}.Pull(trx)
		app := models.App{Id: vm.AppId}.Pull(trx)
		if app.OwnerId != state.Info().UserId() {
			return nil, errors.New("you are not owner of this machine")
		}
		if err := a.App.Tools().File().DeleteFileFromGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/users/"+vm.MachineId, input.EntityId, true); err != nil {
			log.Println(err)
			return nil, err
		}
	}
	return map[string]any{}, nil
}

// UploadPointEntity /storage/uploadPointEntity check [ true true true ] access [ true false false false POST ]
func (a *Actions) UploadPointEntity(state state.IState, input inputs_storage.UploadPointEntityInput) (any, error) {
	trx := state.Trx()
	data, err := base64.StdEncoding.DecodeString(input.Data)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if trx.GetLink("admin::"+state.Info().PointId()+"::"+state.Info().UserId()) == "" {
		return nil, errors.New("you are not admin")
	}
	if err := a.App.Tools().File().SaveDataToGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/points/"+state.Info().PointId(), data, input.EntityId, true); err != nil {
		log.Println(err)
		return nil, err
	}
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("storage/updatePointEntity", state.Info().PointId(), map[string]any{"pointId": state.Info().PointId(), "entityId": input.EntityId}, true, []string{})
	}, false)
	return map[string]any{}, nil
}

// DeletePointEntity /storage/deletePointEntity check [ true true true ] access [ true false false false POST ]
func (a *Actions) DeletePointEntity(state state.IState, input inputs_storage.DeletePointEntityInput) (any, error) {
	trx := state.Trx()
	if trx.GetLink("admin::"+state.Info().PointId()+"::"+state.Info().UserId()) == "" {
		return nil, errors.New("you are not admin")
	}
	if err := a.App.Tools().File().DeleteFileFromGlobalStorage(a.App.Tools().Storage().StorageRoot()+"/entities/points/"+state.Info().PointId(), input.EntityId, true); err != nil {
		log.Println(err)
		return nil, err
	}
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("storage/updatePointEntity", state.Info().PointId(), map[string]any{"pointId": state.Info().PointId(), "entityId": input.EntityId}, true, []string{})
	}, false)
	return map[string]any{}, nil
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
