package wasm

import "kasper/src/abstract/models/worker"

type IWasm interface {
	Assign(machineId string)
	ExecuteChainTrxsGroup(trxs []*worker.Trx)
	ExecuteChainEffects(effects string)
	WasmCallback(dataRaw string) string
	
}
