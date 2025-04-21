package elpis

/*
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

	inputs_topics "kasper/src/shell/api/inputs/topics"
)

type Elpis struct {
	app         abstract.ICore
	logger      *modulelogger.Logger
	storageRoot string
	storage     adapters.IStorage
}

func (wm *Elpis) Assign(machineId string) {
	toolbox := abstract.UseToolbox[toolboxL1.IToolboxL1](wm.app.Get(1).Tools())
	toolbox.Signaler().ListenToSingle(&module_model.Listener{
		Id: machineId,
		Signal: func(a any) {
			astPath := C.CString(toolbox.Storage().StorageRoot() + "/machines/" + machineId + "/module")
			data := string(a.([]byte))
			dataParts := strings.Split(data, " ")
			if dataParts[1] == "topics/send" {
				data = data[len(dataParts[0])+1+len(dataParts[1])+1:]
				var inp inputs_topics.SendInput
				e := json.Unmarshal([]byte(data), &inp)
				if e != nil {
					log.Println(e)
				}
				spaceId := C.CString(inp.SpaceId)
				topicId := C.CString(inp.TopicId)
				memberId := C.CString(inp.MemberId)
				recvId := C.CString(inp.RecvId)
				sendType := C.CString(inp.Type)
				inputData := C.CString(inp.Data)
				C.runVm(astPath, sendType, spaceId, topicId, memberId, recvId, inputData)
			}
		},
	})
}

func (wm *Elpis) ExecuteChainTrxsGroup([]*worker.Trx) {

}

func (wm *Elpis) ElpisCallback(dataRaw string) string {
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
		text, err := checkField(input, "text", "")
		if err != nil {
			log.Println(err)
			return err.Error()
		}
		log.Println("elpis vm:", text)
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

func NewElpis(core abstract.ICore, logger *modulelogger.Logger, storageRoot string, storage adapters.IStorage) *Elpis {
	storage.AutoMigrate(&adapters_model.DataUnit{})
	wm := &Elpis{
		app:         core,
		logger:      logger,
		storageRoot: storageRoot,
		storage:     storage,
	}
	return wm
}
