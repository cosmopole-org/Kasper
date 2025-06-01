package wasm

import "kasper/src/abstract/models/worker"

type IWasm interface {
	Assign(machineId string)
	RunVm(machineId string, pointId string, data string)
	ExecuteChainTrxsGroup(trxs []*worker.Trx)
	ExecuteChainEffects(effects string)
	WasmCallback(dataRaw string) string
}
