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
	inputs_storage "kasper/src/shell/api/inputs/storage"
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
		Signal: func(a any) {
			machId := C.CString(machineId)
			astPath := C.CString(wm.app.Tools().Storage().StorageRoot() + "/machines/" + machineId + "/module")
			data := string(a.([]byte))
			dataParts := strings.Split(data, " ")
			if dataParts[1] == "topics/send" {
				data = data[len(dataParts[0])+1+len(dataParts[1])+1:]
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

func (wm *Wasm) WasmCallback(dataRaw string) string {
	log.Println(dataRaw)
	data := map[string]any{}
	err := json.Unmarshal([]byte(dataRaw), &data)
	if err != nil {
		log.Println(err)
		return err.Error()
	}
	key, err := checkField[string](data, "key", "")
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
		outputFile, err := wm.docker.RunContainer(machineId, pointId, imageName, finalInputFiles)
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
		k, err := checkField(input, "key", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		pointId, err := checkField(input, "pointId", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
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
		pack, err := checkField(input, "packet", "{}")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		var data []byte
		if isFile {
			if wm.file.CheckFileFromStorage(wm.storageRoot, pointId, pack) {
				b, err := wm.file.ReadFileFromStorage(wm.storageRoot, pointId, pack)
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
			if k == "/storage/uploadData" {
				data, _ = json.Marshal(inputs_storage.UploadDataInput{
					Data:    base64.StdEncoding.EncodeToString(data),
					PointId: pointId,
				})
			}
			wm.app.ExecBaseRequestOnChain(k, data, "#appletsign", machineId, func(b []byte, i int, err error) {
				if err != nil {
					log.Println(err)
					return
				}
				result = b
				outputCnan <- 1
			})
		} else {
			wm.app.ExecAppletRequestOnChain(pointId, targetMachineId, k, data, "#appletsign", machineId, func(b []byte, i int, err error) {
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
