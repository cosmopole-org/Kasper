package wasm

/*
 #cgo CXXFLAGS: -std=c++17
 #cgo LDFLAGS: -lrocksdb -lpthread -lz -lsnappy -lzstd -llz4 -lbz2 -lwasmedge -static-libgcc -static-libstdc++

 #include "main.h"
*/
import "C"

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"kasper/src/abstract/adapters/docker"
	"kasper/src/abstract/adapters/file"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/models/update"
	"kasper/src/abstract/models/worker"
	"kasper/src/abstract/state"
	"kasper/src/core/module/actor/model/base"
	inputs_points "kasper/src/shell/api/inputs/points"
	inputs_storage "kasper/src/shell/api/inputs/storage"
	"kasper/src/shell/api/model"
	updates_points "kasper/src/shell/api/updates/points"
	"log"
	"strings"
)

type Wasm struct {
	app         core.ICore
	storageRoot string
	storage     storage.IStorage
	docker      docker.IDocker
	file        file.IFile
}

func (wm *Wasm) Assign(machineId string) {
	wm.app.Tools().Signaler().ListenToSingle(&signaler.Listener{
		Id: machineId,
		Signal: func(key string, a any) {
			machId := C.CString(machineId)
			astPath := C.CString(wm.app.Tools().Storage().StorageRoot() + "/machines/" + machineId + "/module")
			data := string(a.([]byte))
			if key == "points/signal" {
				input := C.CString(data)
				C.wasmRunVm(astPath, input, machId)
			}
		},
	})
}

func (wm *Wasm) ExecuteChainTrxsGroup(trxs []*worker.Trx) {
	b, e := json.Marshal(trxs)
	if e != nil {
		log.Println(e)
		return
	}
	input := C.CString(string(b))
	astStorePath := C.CString(wm.app.Tools().Storage().StorageRoot() + "/machines")
	C.wasmRunTrxs(astStorePath, input)
}

func (wm *Wasm) ExecuteChainEffects(effects string) {
	effectsStr := C.CString(effects)
	C.wasmRunEffects(effectsStr)
}

type ChainDbOp struct {
	OpType string `json:"opType"`
	Key    string `json:"key"`
	Val    string `json:"val"`
}

func (wm *Wasm) RunVm(machineId string, pointId string, data string) {
	point := model.Point{Id: pointId}
	isMemberOfPoint := false
	wm.app.ModifyState(true, func(trx trx.ITrx) {
		point.Pull(trx)
		isMemberOfPoint = (trx.GetLink("memberof::"+machineId+"::"+pointId) == "true")
	})
	if !isMemberOfPoint {
		return
	}
	machId := C.CString(machineId)
	astPath := C.CString(wm.app.Tools().Storage().StorageRoot() + "/machines/" + machineId + "/module")
	b, _ := json.Marshal(updates_points.Send{User: model.User{}, Point: point, Action: "single", Data: data})
	input := C.CString(string(b))
	C.wasmRunVm(astPath, input, machId)
}

func (wm *Wasm) WasmCallback(dataRaw string) string {
	log.Println(dataRaw)
	data := map[string]any{}
	err := json.Unmarshal([]byte(dataRaw), &data)
	if err != nil {
		log.Println(err)
		return err.Error()
	}
	key, err := checkField(data, "key", "")
	if err != nil {
		log.Println(err)
		return err.Error()
	}
	input, err := checkField[map[string]any](data, "input", nil)
	if err != nil {
		log.Println(err)
		return err.Error()
	}
	if key == "runDocker" {
		machineId, err := checkField(input, "machineId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		pointId, err := checkField(input, "pointId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		found := false
		wm.app.ModifyState(true, func(trx trx.ITrx) {
			if trx.GetLink("member::"+pointId+"::"+machineId) == "true" {
				found = true
			}
		})
		if !found {
			err := errors.New("access denied")
			log.Println(err)
			return err.Error()
		}
		inputFilesStr, err := checkField(input, "inputFiles", "{}")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		inputFiles := map[string]string{}
		err = json.Unmarshal([]byte(inputFilesStr), &inputFiles)
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		finalInputFiles := map[string]string{}
		for k, v := range inputFiles {
			if !wm.file.CheckFileFromStorage(wm.storageRoot, pointId, k) {
				err := errors.New("input file does not exist")
				log.Println(err)
				return err.Error()
			}
			path := fmt.Sprintf("%s/files/%s/%s", wm.storageRoot, pointId, k)
			finalInputFiles[path] = v
		}
		imageName, err := checkField(input, "imageName", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		containerName, err := checkField(input, "containerName", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		outputFile, err := wm.docker.RunContainer(machineId, pointId, imageName, containerName, finalInputFiles)
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		if outputFile != nil {
			str, err := json.Marshal(outputFile)
			if err != nil {
				log.Println(err)
				return err.Error()
			}
			return string(str)
		}
	} else if key == "execDocker" {
		machineId, err := checkField(input, "machineId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		imageName, err := checkField(input, "imageName", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		containerName, err := checkField(input, "containerName", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		command, err := checkField(input, "command", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		output, err := wm.docker.ExecContainer(machineId, imageName, containerName, command)
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		return output
	} else if key == "log" {
		_, err := checkField(input, "text", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		// log.Println("elpis vm:", text)
	} else if key == "submitOnchainResponse" {
		callbackId, err := checkField(input, "callbackId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		pack, err := checkField(input, "packet", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		changes, err := checkField(input, "changes", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		resCode, err := checkField[float64](input, "resCode", 0)
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		e, err := checkField(input, "error", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		wm.app.ExecAppletResponseOnChain(callbackId, []byte(pack), "#appletsign", int(resCode), e, []update.Update{{Val: []byte("applet: " + changes)}})
	} else if key == "submitOnchainTrx" {
		machineId, err := checkField(input, "machineId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		targetMachineId, err := checkField(input, "targetMachineId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		kRaw, err := checkField(input, "key", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		kParts := strings.Split(kRaw, "|")
		dstPointId := kParts[0]
	    srcPointId, err := checkField(input, "pointId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		k := kParts[1]
		isFile, err := checkField(input, "isFile", false)
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		isBase, err := checkField(input, "isBase", false)
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		tag, err := checkField(input, "tag", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		pack, err := checkField(input, "packet", "{}")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		var data []byte
		if isFile {
			if wm.file.CheckFileFromStorage(wm.storageRoot, srcPointId, pack) {
				b, err := wm.file.ReadFileFromStorage(wm.storageRoot, srcPointId, pack)
				if err != nil {
					log.Println(err)
					return err.Error()
				}
				data = b
			}
		} else {
			data = []byte(pack)
		}

		result := []byte("{}")
		outputCnan := make(chan int)
		if isBase {
			if k == "/storage/upload" {
				data, _ = json.Marshal(inputs_storage.UploadDataInput{
					Data:    base64.StdEncoding.EncodeToString(data),
					PointId: dstPointId,
				})
			}
			wm.app.ExecBaseRequestOnChain(k, data, "#appletsign", machineId, tag, func(b []byte, i int, err error) {
				if err != nil {
					log.Println(err)
					return
				}
				result = b
				outputCnan <- 1
			})
		} else {
			wm.app.ExecAppletRequestOnChain(dstPointId, targetMachineId, k, data, "#appletsign", machineId, tag, func(b []byte, i int, err error) {
				if err != nil {
					log.Println(err)
					return
				}
				result = b
				outputCnan <- 1
			})
		}
		<-outputCnan
		return string(result)
	} else if key == "plantTrigger" {
		count, err := checkField(input, "count", float64(0))
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		machineId, err := checkField(input, "machineId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		tag, err := checkField(input, "tag", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		pointId, err := checkField(input, "pointId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		data, err := checkField(input, "input", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		wm.app.PlantChainTrigger(int(count), machineId, tag, machineId, pointId, data)
	} else if key == "signalPoint" {
		machineId, err := checkField(input, "machineId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		typ, err := checkField(input, "type", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		pointId, err := checkField(input, "pointId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		userId, err := checkField(input, "userId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		data, err := checkField(input, "data", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		wm.app.ModifyStateSecurly(false, base.NewInfo(machineId, pointId), func(s state.IState) {
			wm.app.Actor().FetchAction("/points/signal").Act(s, inputs_points.SignalInput{
				Type: typ,
				Data: data,
				PointId: pointId,
				UserId: userId,
			})
		})
	}

	return "{}"
}

func checkField[T any](input map[string]any, fieldName string, defVal T) (T, error) {
	fRaw, ok := input[fieldName]
	if !ok {
		return defVal, errors.New("{\"error\":1}}")
	}
	f, ok := fRaw.(T)
	if !ok {
		return defVal, errors.New("{\"error\":2}}")
	}
	return f, nil
}

func NewWasm(core core.ICore, storageRoot string, storage storage.IStorage, kvDbPath string, docker docker.IDocker, file file.IFile) *Wasm {
	wm := &Wasm{
		app:         core,
		storageRoot: storageRoot,
		storage:     storage,
		docker:      docker,
		file:        file,
	}
	C.init(C.CString(kvDbPath))
	return wm
}
