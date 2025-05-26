package core

import (
	"kasper/src/abstract/adapters/tools"
	"kasper/src/abstract/models/action"
	"kasper/src/abstract/models/chain"
	"kasper/src/abstract/models/info"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/models/update"
	"kasper/src/abstract/state"
	"kasper/src/hashgraph"
	"kasper/src/proxy"
	"log"
	"strings"
	hgstate "kasper/src/node/state"
)

type EmptyPayload struct{}

type ResponseHolder struct {
	Payload []byte
	Effects chain.Effects
}

type ICore interface {
	Id() string
	Gods() []string
	Tools() tools.ITools
	Actor() action.IActor
	Load([]string, map[string]interface{})
	PlantChainTrigger(count int, userId string, tag string, machineId string, pointId string, input string)
	ExecAppletRequestOnChain(pointId string, machineId string, key string, payload []byte, signature string, userId string, tag string, callback func([]byte, int, error))
	ExecAppletResponseOnChain(callbackId string, packet []byte, signature string, resCode int, e string, updates []update.Update)
	ExecBaseRequestOnChain(key string, payload []byte, signature string, userId string, tag string, callback func([]byte, int, error))
	ExecBaseResponseOnChain(callbackId string, packet []byte, signature string, resCode int, e string, updates []update.Update, tag string, toUserId string)
	OnChainPacket(typ string, trxPayload []byte)
	AppPendingTrxs()
	Run()
	NewHgHandler() *HgHandler
	IpAddr() string
	ModifyState(bool, func(trx.ITrx))
	ModifyStateSecurlyWithSource(readonly bool, info info.IInfo, src string, fn func(state.IState))
	ModifyStateSecurly(readonly bool, info info.IInfo, fn func(state.IState))
	SignPacket(data []byte) string
}

type HgHandler struct {
	State hgstate.State
	Sigma ICore
}

func (p *HgHandler) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	for _, trx := range block.Transactions() {
		firstIndex := strings.Index(string(trx), "::")
		log.Println(string(trx))
		p.Sigma.OnChainPacket(string(trx[:firstIndex]), trx[firstIndex+2:])
	}

	p.Sigma.AppPendingTrxs()

	receipts := []hashgraph.InternalTransactionReceipt{}
	for _, it := range block.InternalTransactions() {
		receipts = append(receipts, it.AsAccepted())
	}
	response := proxy.CommitResponse{
		StateHash:                   []byte("statehash"),
		InternalTransactionReceipts: receipts,
	}
	return response, nil
}

func (p *HgHandler) StateChangeHandler(state hgstate.State) error {
	p.State = state
	return nil
}

func (p *HgHandler) SnapshotHandler(blockIndex int) ([]byte, error) {
	return []byte("statehash"), nil
}

func (p *HgHandler) RestoreHandler(snapshot []byte) ([]byte, error) {
	return []byte("statehash"), nil
}
