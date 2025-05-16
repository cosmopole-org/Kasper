package wasm

// package wasmold

// import (
// 	"encoding/binary"
// 	"encoding/json"
// 	"fmt"
// 	"kasper/src/abstract"
// 	inputs_topics "kasper/src/shell/api/inputs/topics"
// 	moduleactormodel "kasper/src/core/module/actor/model"
// 	modulelogger "kasper/src/core/module/logger"
// 	"kasper/src/shell/layer1/adapters"
// 	adapters_model "kasper/src/shell/layer1/adapters/model"
// 	module_model "kasper/src/shell/layer1/model"
// 	statel1 "kasper/src/shell/layer1/module/state"
// 	toolboxL1 "kasper/src/shell/layer1/module/toolbox"
// 	"kasper/src/shell/utils/future"
// 	"log"
// 	"math/rand"
// 	"os"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/second-state/WasmEdge-go/wasmedge"
// )

// type Wasm struct {
// 	app      abstract.ICore
// 	logger         *modulelogger.Logger
// 	storageRoot    string
// 	storage        adapters.IStorage
// 	PluginVms      map[string]*wasmedge.VM
// 	PluginVmsByKey map[string]*wasmedge.VM
// 	PluginMetas    map[string]abstract.IAction
// }

// func (wm *Wasm) prepareVm(wasmFilePath string, machineId string) (*wasmedge.VM, error) {
// 	var conf = wasmedge.NewConfigure(wasmedge.REFERENCE_TYPES)
// 	conf.AddConfig(wasmedge.WASI)
// 	vm := wasmedge.NewVMWithConfig(conf)
// 	var wasi = vm.GetImportModule(wasmedge.WASI)
// 	wasi.InitWasi(
// 		os.Args[1:],     // The args
// 		os.Environ(),    // The envs
// 		[]string{".:."}, // The mapping directories
// 	)

// 	obj := wasmedge.NewModule("env")

// 	funcSqlType := wasmedge.NewFunctionType(
// 		[]*wasmedge.ValType{
// 			wasmedge.NewValTypeI32(),
// 		},
// 		[]*wasmedge.ValType{
// 			wasmedge.NewValTypeI32(),
// 		})
// 	h := &vmHost{vm: vm, wasmTool: wm, machineId: machineId}
// 	hostSql := wasmedge.NewFunction(funcSqlType, h.message, nil, 0)
// 	obj.AddFunction("message", hostSql)

// 	funcLogType := wasmedge.NewFunctionType(
// 		[]*wasmedge.ValType{
// 			wasmedge.NewValTypeI32(),
// 		},
// 		[]*wasmedge.ValType{})
// 	hostLog := wasmedge.NewFunction(funcLogType, h.logData, nil, 0)
// 	obj.AddFunction("logData", hostLog)

// 	err5 := vm.RegisterModule(obj)
// 	if err5 != nil {
// 		wm.logger.Println("failed to register wasm")
// 		return nil, err5
// 	}
// 	err4 := vm.LoadWasmFile(wasmFilePath)
// 	if err4 != nil {
// 		wm.logger.Println("failed to load wasm")
// 		return nil, err4
// 	}
// 	err3 := vm.Validate()
// 	if err3 != nil {
// 		wm.logger.Println("failed to validate wasm")
// 		return nil, err3
// 	}
// 	err2 := vm.Instantiate()
// 	if err2 != nil {
// 		wm.logger.Println("failed to instantiate wasm")
// 		return nil, err2
// 	}
// 	_, err1 := vm.Execute("_start")
// 	if err1 != nil {
// 		wm.logger.Println(err1)
// 		return nil, err1
// 	}
// 	return vm, nil
// }

// func deallocteVals(vm *wasmedge.VM, pointer any) {
// 	// _, err2 := vm.Execute("free", pointer)
// 	// if err2 != nil {
// 	// 	log.Println(err2)
// 	// }
// }

// func (wm *Wasm) Assign(machineId string) {
// 	toolbox := abstract.UseToolbox[toolboxL1.IToolboxL1](wm.app.Get(1).Tools())
// 	toolbox.Signaler().ListenToSingle(&module_model.Listener{
// 		Id: machineId,
// 		Signal: func(a any) {

// 			vm, errVm := wm.prepareVm(toolbox.Storage().StorageRoot()+"/machines/"+machineId+"/module", machineId)
// 			if errVm != nil {
// 				log.Println(errVm)
// 				return
// 			}
// 			str, ok := a.([]byte)
// 			var body string
// 			var key string
// 			if ok {
// 				parts := strings.Split(string(str), " ")
// 				key = parts[1]
// 				body = string(str)[len(parts[0])+1+len(parts[1])+1:]
// 			} else {
// 				str, errSerialize := json.Marshal(a)
// 				if errSerialize != nil {
// 					log.Println(errSerialize)
// 					return
// 				}
// 				body = string(str)
// 			}

// 			var lengthOfSubject = len(body)
// 			var lengthOfKey = len(key)

// 			if lengthOfKey == 0 || lengthOfSubject == 0 {
// 				log.Println("body or key can not be empty")
// 				return
// 			}

// 			keyAllocateResult, allocErr1 := vm.Execute("malloc", int32(lengthOfKey+1))
// 			if allocErr1 != nil {
// 				log.Println(allocErr1)
// 				return
// 			}
// 			keyiInputPointer := keyAllocateResult[0].(int32)
// 			allocateResult, allocErr2 := vm.Execute("malloc", int32(lengthOfSubject+1))
// 			if allocErr2 != nil {
// 				log.Println(allocErr2)
// 				return
// 			}
// 			inputPointer := allocateResult[0].(int32)

// 			mod := vm.GetActiveModule()
// 			mem := mod.FindMemory("memory")
// 			keyMemData, getDataErr1 := mem.GetData(uint(keyiInputPointer), uint(lengthOfKey+1))
// 			if getDataErr1 != nil {
// 				log.Println(getDataErr1)
// 				return
// 			}
// 			copy(keyMemData, key)
// 			memData, getDataErr2 := mem.GetData(uint(inputPointer), uint(lengthOfSubject+1))
// 			if getDataErr2 != nil {
// 				log.Println(getDataErr2)
// 				return
// 			}
// 			copy(memData, body)

// 			keyMemData[lengthOfKey] = 0
// 			memData[lengthOfSubject] = 0

// 			greetResult, runErr := vm.Execute("run", int32(lengthOfKey), keyiInputPointer, int32(lengthOfSubject), inputPointer)
// 			if runErr != nil {
// 				log.Println(runErr)

// 				deallocteVals(vm, inputPointer)
// 				return
// 			}
// 			if len(greetResult) == 0 {
// 				log.Println("output is empty")

// 				deallocteVals(vm, inputPointer)
// 				return
// 			}
// 			outputPointer := greetResult[0].(int32)

// 			getDataErr2 = nil
// 			memData, getDataErr2 = mem.GetData(uint(outputPointer), 8)
// 			if getDataErr2 != nil {
// 				log.Println(getDataErr2)

// 				deallocteVals(vm, inputPointer)

// 				return
// 			}
// 			resultPointer := binary.LittleEndian.Uint32(memData[:4])
// 			resultLength := binary.LittleEndian.Uint32(memData[4:])

// 			// Read the result of the `greet` function.
// 			getDataErr2 = nil
// 			memData, getDataErr2 = mem.GetData(uint(resultPointer), uint(resultLength))
// 			if getDataErr2 != nil {
// 				log.Println(getDataErr2)

// 				deallocteVals(vm, inputPointer)

// 				return
// 			}

// 			deallocteVals(vm, inputPointer)

// 			var output map[string]interface{}
// 			err := json.Unmarshal(memData, &output)
// 			if err != nil {
// 				wm.logger.Println(err)
// 				return
// 			}
// 		},
// 	})
// }

// func (wm *Wasm) Plug(wasmFilePath string, key string) {

// 	vm, err := wm.prepareVm(wasmFilePath, key)
// 	if err != nil {
// 		wm.logger.Println(err)
// 		return
// 	}

// 	oldVm, ok := wm.PluginVmsByKey[key]
// 	if ok {
// 		oldVm.Release()
// 	}
// 	wm.PluginVmsByKey[key] = vm

// }

// type vmHost struct {
// 	machineId string
// 	wasmTool  *Wasm
// 	vm        *wasmedge.VM
// }

// func (h *vmHost) logData(_ interface{}, callframe *wasmedge.CallingFrame, params []interface{}) ([]interface{}, wasmedge.Result) {
// 	h.wasmTool.logger.Println(h.remotePtrToString(params[0].(int32), callframe))
// 	return []interface{}{}, wasmedge.Result_Success
// }

// type WasmPacket struct {
// 	host *vmHost
// 	data map[string]any
// }

// var wasmPipe = make(chan WasmPacket, 1)

// func (h *Wasm) HandleWasmRequest() {

// 	for {
// 		var wp = <-wasmPipe

// 		packet := wp.data
// 		h := wp.host

// 		time.Sleep(time.Duration(1) * time.Second)

// 		switch packet["key"].(string) {
// 		case "math/genRandGroup":
// 			{
// 				callbackId := packet["callbackId"].(string)
// 				max := packet["max"].(string)
// 				maxNum, err := strconv.ParseInt(max, 10, 64)
// 				if err != nil {
// 					log.Println(err)
// 					break
// 				}
// 				rndArr := []string{}
// 				for i := 0; i < int(maxNum); i++ {
// 					rndArr = append(rndArr, fmt.Sprintf("%d", rand.Intn(int(maxNum))))
// 				}

// 				future.Async(func() {

// 					time.Sleep(time.Duration(1000) * time.Millisecond)

// 					var body string
// 					var key = "response/math/genRandGroup"

// 					str, errSerialize := json.Marshal(map[string]any{
// 						"numbers":    rndArr,
// 						"callbackId": callbackId,
// 					})
// 					if errSerialize != nil {
// 						log.Println(errSerialize)
// 						return
// 					}
// 					body = string(str)

// 					var lengthOfSubject = len(body)
// 					var lengthOfKey = len(key)

// 					if lengthOfKey == 0 || lengthOfSubject == 0 {
// 						log.Println("body or key can not be empty")
// 						return
// 					}

// 					vm := h.vm

// 					keyAllocateResult, allocErr1 := vm.Execute("malloc", int32(lengthOfKey+1))
// 					if allocErr1 != nil {
// 						log.Println(allocErr1)
// 						return
// 					}
// 					keyiInputPointer := keyAllocateResult[0].(int32)
// 					allocateResult, allocErr2 := vm.Execute("malloc", int32(lengthOfSubject+1))
// 					if allocErr2 != nil {
// 						log.Println(allocErr2)
// 						return
// 					}
// 					inputPointer := allocateResult[0].(int32)

// 					// Write the subject into the memory.
// 					mod := vm.GetActiveModule()
// 					mem := mod.FindMemory("memory")
// 					keyMemData, getDataErr1 := mem.GetData(uint(keyiInputPointer), uint(lengthOfKey+1))
// 					if getDataErr1 != nil {
// 						log.Println(getDataErr1)
// 						return
// 					}
// 					copy(keyMemData, key)
// 					memData, getDataErr2 := mem.GetData(uint(inputPointer), uint(lengthOfSubject+1))
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)
// 						return
// 					}
// 					copy(memData, body)

// 					// C-string terminates by NULL.
// 					keyMemData[lengthOfKey] = 0
// 					memData[lengthOfSubject] = 0

// 					// Run the `greet` function. Given the pointer to the subject.
// 					greetResult, runErr := vm.Execute("run", int32(lengthOfKey), keyiInputPointer, int32(lengthOfSubject), inputPointer)
// 					if runErr != nil {
// 						log.Println(runErr)

// 						deallocteVals(vm, inputPointer)
// 						return
// 					}
// 					if len(greetResult) == 0 {
// 						log.Println("output is empty")

// 						deallocteVals(vm, inputPointer)
// 						return
// 					}
// 					outputPointer := greetResult[0].(int32)

// 					getDataErr2 = nil
// 					memData, getDataErr2 = mem.GetData(uint(outputPointer), 8)
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)

// 						deallocteVals(vm, inputPointer)

// 						return
// 					}
// 					resultPointer := binary.LittleEndian.Uint32(memData[:4])
// 					resultLength := binary.LittleEndian.Uint32(memData[4:])

// 					// Read the result of the `greet` function.
// 					getDataErr2 = nil
// 					memData, getDataErr2 = mem.GetData(uint(resultPointer), uint(resultLength))
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)

// 						deallocteVals(vm, inputPointer)

// 						return
// 					}

// 					deallocteVals(vm, inputPointer)

// 					var output map[string]interface{}
// 					err := json.Unmarshal(memData, &output)
// 					if err != nil {
// 						log.Println(err)
// 						return
// 					}
// 				})

// 				break
// 			}
// 		case "math/genRand":
// 			{
// 				callbackId := packet["callbackId"].(string)
// 				max := packet["max"].(string)
// 				maxNum, err := strconv.ParseInt(max, 10, 64)
// 				if err != nil {
// 					log.Println(err)
// 					break
// 				}
// 				rnd := fmt.Sprintf("%d", rand.Intn(int(maxNum)))

// 				future.Async(func() {

// 					time.Sleep(time.Duration(250) * time.Millisecond)

// 					var body string
// 					var key = "response/math/genRand"

// 					str, errSerialize := json.Marshal(map[string]any{
// 						"number":     rnd,
// 						"callbackId": callbackId,
// 					})
// 					if errSerialize != nil {
// 						log.Println(errSerialize)
// 						return
// 					}
// 					body = string(str)

// 					var lengthOfSubject = len(body)
// 					var lengthOfKey = len(key)

// 					if lengthOfKey == 0 || lengthOfSubject == 0 {
// 						log.Println("body or key can not be empty")
// 						return
// 					}

// 					vm := h.vm

// 					keyAllocateResult, allocErr1 := vm.Execute("malloc", int32(lengthOfKey+1))
// 					if allocErr1 != nil {
// 						log.Println(allocErr1)
// 						return
// 					}
// 					keyiInputPointer := keyAllocateResult[0].(int32)
// 					allocateResult, allocErr2 := vm.Execute("malloc", int32(lengthOfSubject+1))
// 					if allocErr2 != nil {
// 						log.Println(allocErr2)
// 						return
// 					}
// 					inputPointer := allocateResult[0].(int32)

// 					// Write the subject into the memory.
// 					mod := vm.GetActiveModule()
// 					mem := mod.FindMemory("memory")
// 					keyMemData, getDataErr1 := mem.GetData(uint(keyiInputPointer), uint(lengthOfKey+1))
// 					if getDataErr1 != nil {
// 						log.Println(getDataErr1)
// 						return
// 					}
// 					copy(keyMemData, key)
// 					memData, getDataErr2 := mem.GetData(uint(inputPointer), uint(lengthOfSubject+1))
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)
// 						return
// 					}
// 					copy(memData, body)

// 					// C-string terminates by NULL.
// 					keyMemData[lengthOfKey] = 0
// 					memData[lengthOfSubject] = 0

// 					// Run the `greet` function. Given the pointer to the subject.
// 					greetResult, runErr := vm.Execute("run", int32(lengthOfKey), keyiInputPointer, int32(lengthOfSubject), inputPointer)
// 					if runErr != nil {
// 						log.Println(runErr)

// 						deallocteVals(vm, inputPointer)
// 						return
// 					}
// 					if len(greetResult) == 0 {
// 						log.Println("output is empty")

// 						deallocteVals(vm, inputPointer)
// 						return
// 					}
// 					outputPointer := greetResult[0].(int32)

// 					getDataErr2 = nil
// 					memData, getDataErr2 = mem.GetData(uint(outputPointer), 8)
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)

// 						deallocteVals(vm, inputPointer)

// 						return
// 					}
// 					resultPointer := binary.LittleEndian.Uint32(memData[:4])
// 					resultLength := binary.LittleEndian.Uint32(memData[4:])

// 					// Read the result of the `greet` function.
// 					getDataErr2 = nil
// 					memData, getDataErr2 = mem.GetData(uint(resultPointer), uint(resultLength))
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)

// 						deallocteVals(vm, inputPointer)

// 						return
// 					}

// 					deallocteVals(vm, inputPointer)

// 					var output map[string]interface{}
// 					err := json.Unmarshal(memData, &output)
// 					if err != nil {
// 						log.Println(err)
// 						return
// 					}
// 				})

// 				break
// 			}
// 		case "database/save":
// 			{
// 				id := packet["id"].(string)
// 				topicId := packet["topicId"].(string)
// 				memberId := packet["memberId"].(string)
// 				data := packet["data"].(string)
// 				h.wasmTool.storage.DoTrx(func(trx adapters.ITrx) error {
// 					dts := adapters_model.DataUnit{Id: id}
// 					err := trx.Db().Model(&adapters_model.DataUnit{}).Where("topic_id = ?", topicId).Where("member_id = ?", memberId).First(&dts).Error
// 					dts = adapters_model.DataUnit{Id: id, Data: data, TopicId: topicId, MemberId: memberId}
// 					trx.ClearError()
// 					if err != nil {
// 						trx.Db().Create(&dts)
// 					} else {
// 						trx.Db().Save(&dts)
// 					}
// 					return nil
// 				})
// 				break
// 			}
// 		case "database/fetch":
// 			{
// 				id := packet["id"].(string)
// 				topicId := packet["topicId"].(string)
// 				memberId := packet["memberId"].(string)
// 				dts := adapters_model.DataUnit{Id: id}
// 				h.wasmTool.storage.DoTrx(func(trx adapters.ITrx) error {
// 					return trx.Db().Model(&adapters_model.DataUnit{}).Where("topic_id = ?", topicId).Where("member_id = ?", memberId).First(&dts).Error
// 				})

// 				log.Println()
// 				log.Println("dts:")
// 				log.Println(dts)
// 				log.Println()

// 				future.Async(func() {

// 					time.Sleep(time.Duration(100) * time.Millisecond)

// 					var body string
// 					var key = "response/database/fetch"

// 					str, errSerialize := json.Marshal(map[string]any{
// 						"data":     dts.Data,
// 						"topicId":  topicId,
// 						"memberId": memberId,
// 						"id":       dts.Id,
// 					})
// 					if errSerialize != nil {
// 						log.Println(errSerialize)
// 						return
// 					}
// 					body = string(str)

// 					var lengthOfSubject = len(body)
// 					var lengthOfKey = len(key)

// 					if lengthOfKey == 0 || lengthOfSubject == 0 {
// 						log.Println("body or key can not be empty")
// 						return
// 					}

// 					vm := h.vm

// 					keyAllocateResult, allocErr1 := vm.Execute("malloc", int32(lengthOfKey+1))
// 					if allocErr1 != nil {
// 						log.Println(allocErr1)
// 						return
// 					}
// 					keyiInputPointer := keyAllocateResult[0].(int32)
// 					allocateResult, allocErr2 := vm.Execute("malloc", int32(lengthOfSubject+1))
// 					if allocErr2 != nil {
// 						log.Println(allocErr2)
// 						return
// 					}
// 					inputPointer := allocateResult[0].(int32)

// 					// Write the subject into the memory.
// 					mod := vm.GetActiveModule()
// 					mem := mod.FindMemory("memory")
// 					keyMemData, getDataErr1 := mem.GetData(uint(keyiInputPointer), uint(lengthOfKey+1))
// 					if getDataErr1 != nil {
// 						log.Println(getDataErr1)
// 						return
// 					}
// 					copy(keyMemData, key)
// 					memData, getDataErr2 := mem.GetData(uint(inputPointer), uint(lengthOfSubject+1))
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)
// 						return
// 					}
// 					copy(memData, body)

// 					// C-string terminates by NULL.
// 					keyMemData[lengthOfKey] = 0
// 					memData[lengthOfSubject] = 0

// 					// Run the `greet` function. Given the pointer to the subject.
// 					greetResult, runErr := vm.Execute("run", int32(lengthOfKey), keyiInputPointer, int32(lengthOfSubject), inputPointer)
// 					if runErr != nil {
// 						log.Println(runErr)

// 						deallocteVals(vm, inputPointer)
// 						return
// 					}
// 					if len(greetResult) == 0 {
// 						log.Println("output is empty")

// 						deallocteVals(vm, inputPointer)
// 						return
// 					}
// 					outputPointer := greetResult[0].(int32)

// 					getDataErr2 = nil
// 					memData, getDataErr2 = mem.GetData(uint(outputPointer), 8)
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)

// 						deallocteVals(vm, inputPointer)

// 						return
// 					}
// 					resultPointer := binary.LittleEndian.Uint32(memData[:4])
// 					resultLength := binary.LittleEndian.Uint32(memData[4:])

// 					// Read the result of the `greet` function.
// 					getDataErr2 = nil
// 					memData, getDataErr2 = mem.GetData(uint(resultPointer), uint(resultLength))
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)

// 						deallocteVals(vm, inputPointer)

// 						return
// 					}

// 					deallocteVals(vm, inputPointer)

// 					var output map[string]interface{}
// 					err := json.Unmarshal(memData, &output)
// 					if err != nil {
// 						log.Println(err)
// 						return
// 					}
// 				})

// 				break
// 			}
// 		case "runtime/delay":
// 			{
// 				future.Async(func() {
// 					delayStr := packet["delay"].(string)
// 					delay, errParse := strconv.ParseInt(delayStr, 10, 64)
// 					if errParse != nil {
// 						log.Println(errParse)
// 						return
// 					}
// 					time.Sleep(time.Duration(delay) * time.Second)

// 					var body string
// 					var key = "runtime/callback"

// 					callbackId := packet["callbackId"].(string)

// 					str, errSerialize := json.Marshal(map[string]any{
// 						"callbackId": callbackId,
// 					})
// 					if errSerialize != nil {
// 						log.Println(errSerialize)
// 						return
// 					}
// 					body = string(str)

// 					var lengthOfSubject = len(body)
// 					var lengthOfKey = len(key)

// 					if lengthOfKey == 0 || lengthOfSubject == 0 {
// 						log.Println("body or key can not be empty")
// 						return
// 					}

// 					vm := h.vm

// 					keyAllocateResult, allocErr1 := vm.Execute("malloc", int32(lengthOfKey+1))
// 					if allocErr1 != nil {
// 						log.Println(allocErr1)
// 						return
// 					}
// 					keyiInputPointer := keyAllocateResult[0].(int32)
// 					allocateResult, allocErr2 := vm.Execute("malloc", int32(lengthOfSubject+1))
// 					if allocErr2 != nil {
// 						log.Println(allocErr2)
// 						return
// 					}
// 					inputPointer := allocateResult[0].(int32)

// 					// Write the subject into the memory.
// 					mod := vm.GetActiveModule()
// 					mem := mod.FindMemory("memory")
// 					keyMemData, getDataErr1 := mem.GetData(uint(keyiInputPointer), uint(lengthOfKey+1))
// 					if getDataErr1 != nil {
// 						log.Println(getDataErr1)
// 						return
// 					}
// 					copy(keyMemData, key)
// 					memData, getDataErr2 := mem.GetData(uint(inputPointer), uint(lengthOfSubject+1))
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)
// 						return
// 					}
// 					copy(memData, body)

// 					// C-string terminates by NULL.
// 					keyMemData[lengthOfKey] = 0
// 					memData[lengthOfSubject] = 0

// 					// Run the `greet` function. Given the pointer to the subject.
// 					greetResult, runErr := vm.Execute("run", int32(lengthOfKey), keyiInputPointer, int32(lengthOfSubject), inputPointer)
// 					if runErr != nil {
// 						log.Println(runErr)

// 						deallocteVals(vm, inputPointer)
// 						return
// 					}
// 					if len(greetResult) == 0 {
// 						log.Println("output is empty")

// 						deallocteVals(vm, inputPointer)
// 						return
// 					}
// 					outputPointer := greetResult[0].(int32)

// 					getDataErr2 = nil
// 					memData, getDataErr2 = mem.GetData(uint(outputPointer), 8)
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)

// 						deallocteVals(vm, inputPointer)

// 						return
// 					}
// 					resultPointer := binary.LittleEndian.Uint32(memData[:4])
// 					resultLength := binary.LittleEndian.Uint32(memData[4:])

// 					// Read the result of the `greet` function.
// 					getDataErr2 = nil
// 					memData, getDataErr2 = mem.GetData(uint(resultPointer), uint(resultLength))
// 					if getDataErr2 != nil {
// 						log.Println(getDataErr2)

// 						deallocteVals(vm, inputPointer)

// 						return
// 					}

// 					deallocteVals(vm, inputPointer)

// 					var output map[string]interface{}
// 					err := json.Unmarshal(memData, &output)
// 					if err != nil {
// 						log.Println(err)
// 						return
// 					}

// 				})
// 				break
// 			}
// 		case "topics/send":
// 			{
// 				value := packet["value"].(map[string]any)
// 				var machineId = h.machineId
// 				var spaceId = value["spaceId"].(string)
// 				var topicId = value["topicId"].(string)
// 				var memberId = value["memberId"].(string)
// 				var recvId = ""
// 				var recvIdRaw, ok = value["recvId"]
// 				if ok {
// 					recvId = recvIdRaw.(string)
// 				}
// 				var transferType = value["type"].(string)
// 				var dataVal = value["data"].(map[string]any)
// 				dataStr, err2 := json.Marshal(dataVal)
// 				if err2 != nil {
// 					h.wasmTool.logger.Println(err2)
// 				}
// 				var inp = inputs_topics.SendInput{
// 					Type:     transferType,
// 					TopicId:  topicId,
// 					SpaceId:  spaceId,
// 					MemberId: memberId,
// 					RecvId:   recvId,
// 					Data:     string(dataStr),
// 				}
// 				log.Println(inp)
// 				future.Async(func() {

// 					time.Sleep(time.Duration(1000) * time.Millisecond)

// 					stateOfReq := h.wasmTool.app.Get(1).Sb().NewState(moduleactormodel.NewInfo(machineId, spaceId, topicId, memberId)).(*statel1.StateL1)
// 					h.wasmTool.storage.DoTrx(func(trx adapters.ITrx) error {
// 						stateOfReq.SetTrx(trx)
// 						_, res, err := h.wasmTool.app.Get(1).Actor().FetchAction("/topics/send").Act(stateOfReq, inp)
// 						h.wasmTool.logger.Println(res)
// 						return err
// 					})
// 				})
// 				break
// 			}
// 		}
// 	}
// }

// func (h *vmHost) message(_ interface{}, callframe *wasmedge.CallingFrame, params []interface{}) ([]interface{}, wasmedge.Result) {

// 	dataRaw := h.remotePtrToString(params[0].(int32), callframe)
// 	h.wasmTool.logger.Println(dataRaw)
// 	packet := map[string]any{}
// 	err := json.Unmarshal([]byte(dataRaw), &packet)
// 	if err != nil {
// 		h.wasmTool.logger.Println(err)
// 	}

// 	wasmPipe <- WasmPacket{data: packet, host: h}

// 	return []interface{}{interface{}(h.localStringToPtr("response for message", callframe))}, wasmedge.Result_Success
// }

// func (h *vmHost) remotePtrToString(pointer int32, callframe *wasmedge.CallingFrame) string {
// 	mem := callframe.GetMemoryByIndex(0)
// 	memData, _ := mem.GetData(uint(pointer), 8)
// 	resultPointer := binary.LittleEndian.Uint32(memData[:4])
// 	resultLength := binary.LittleEndian.Uint32(memData[4:])
// 	data, _ := mem.GetData(uint(resultPointer), uint(resultLength))
// 	url := make([]byte, resultLength)
// 	copy(url, data)
// 	return string(url)
// }

// func (h *vmHost) localStringToPtr(data string, callframe *wasmedge.CallingFrame) int32 {
// 	mem := callframe.GetMemoryByIndex(0)
// 	data2 := []byte(data)
// 	result, _ := h.vm.Execute("malloc", int32(len(data2)+1))
// 	pointer := result[0].(int32)
// 	m, _ := mem.GetData(uint(pointer), uint(len(data2)+1))
// 	copy(m[:len(data2)], data2)
// 	copy(m[len(data2):], []byte{0})
// 	return pointer
// }

// func NewWasm(core abstract.ICore, logger *modulelogger.Logger, storageRoot string, storage adapters.IStorage) *Wasm {
// 	storage.AutoMigrate(&adapters_model.DataUnit{})
// 	wm := &Wasm{
// 		app:      core,
// 		logger:         logger,
// 		storageRoot:    storageRoot,
// 		storage:        storage,
// 		PluginVms:      make(map[string]*wasmedge.VM),
// 		PluginVmsByKey: make(map[string]*wasmedge.VM),
// 		PluginMetas:    make(map[string]abstract.IAction),
// 	}
// 	wasmedge.SetLogDebugLevel()
// 	future.Async(func() { wm.HandleWasmRequest() })
// 	return wm
// }
