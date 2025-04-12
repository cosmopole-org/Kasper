package wasm

/*
 #cgo CXXFLAGS: -std=c++17
 #cgo LDFLAGS: -lrocksdb -lpthread -lz -lsnappy -lzstd -llz4 -lbz2 -lwasmedge -static-libgcc -static-libstdc++

 #include "main.h"
*/
import "C"

import (
	"encoding/json"
	"errors"
	"kasper/src/abstract"
	"kasper/src/core/module/core/model/worker"
	modulelogger "kasper/src/core/module/logger"
	"kasper/src/shell/layer1/adapters"
	adapters_model "kasper/src/shell/layer1/adapters/model"
	module_model "kasper/src/shell/layer1/model"
	toolboxL1 "kasper/src/shell/layer1/module/toolbox"
	"log"
	"strings"
)

type Wasm struct {
	app         abstract.ICore
	logger      *modulelogger.Logger
	storageRoot string
	storage     adapters.IStorage
}

func (wm *Wasm) Assign(machineId string) {
	toolbox := abstract.UseToolbox[toolboxL1.IToolboxL1](wm.app.Get(1).Tools())
	toolbox.Signaler().ListenToSingle(&module_model.Listener{
		Id: machineId,
		Signal: func(a any) {
			machId := C.CString(machineId)
			astPath := C.CString(toolbox.Storage().StorageRoot() + "/machines/" + machineId + "/module")
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
	toolbox := abstract.UseToolbox[toolboxL1.IToolboxL1](wm.app.Get(1).Tools())
	b, e := json.Marshal(trxs)
	if e != nil {
		log.Println(e)
		return
	}
	input := C.CString(string(b))
	astStorePath := C.CString(toolbox.Storage().StorageRoot() + "/machines")
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
	if key == "log" {
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
		wm.app.ExecAppletResponseOnChain(callbackId, []byte(pack), int(resCode), e, []abstract.Update{{Data: "kvstore: " + changes}}, []abstract.CacheUpdate{})
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
		pack, err := checkField(input, "packet", "{}")
		if err != nil {
			log.Println(err)
			return err.Error()
		}

		result := []byte("{}")
		outputCnan := make(chan int)
		wm.app.ExecAppletRequestOnChain(targetMachineId, k, []byte(pack), machineId, func(b []byte, i int, err error) {
			if err != nil {
				log.Println(err)
				return
			}
			result = b
			outputCnan <- 1
		})
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

func NewWasm(core abstract.ICore, logger *modulelogger.Logger, storageRoot string, storage adapters.IStorage, kvDbPath string) *Wasm {
	storage.AutoMigrate(&adapters_model.DataUnit{})
	wm := &Wasm{
		app:         core,
		logger:      logger,
		storageRoot: storageRoot,
		storage:     storage,
	}
	C.init(C.CString(kvDbPath))
	return wm
}
