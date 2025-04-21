package core

import (
	"encoding/json"
	"kasper/src/abstract/adapters/tools"
	"kasper/src/abstract/models/chain"
	"kasper/src/abstract/models/info"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/models/update"
	"kasper/src/babble"
	"kasper/src/core/module/actor/model/l2"
	"kasper/src/hashgraph"
	"kasper/src/node/state"
	"kasper/src/proxy"
	"log"
)

type EmptyPayload struct{}

type ResponseHolder struct {
	Payload []byte
	Effects chain.Effects
}

type ICore interface {
	Id() string
	Gods() []string
	Tools() *tools.Tools
	Load([]interface{})
	ExecAppletRequestOnChain(pointId string, machineId string, key string, packet []byte, signature string, userId string, callback func([]byte, int, error))
	ExecBaseRequestOnChain(key string, userId string, packet []byte, signature string, callback func([]byte, int, error))
	ExecAppletResponseOnChain(callbackId string, packet []byte, resCode int, e string, updates []update.Update)
	ExecBaseResponseOnChain(callbackId string, packet any, resCode int, e string, updates []update.Update)
	OnChainPacket(packet chain.ChainPacket)
	AppPendingTrxs()
	Chain() *babble.Babble
	Run()
	NewHgHandler() *HgHandler
	IpAddr() string
	ModifyState(bool, func(trx.ITrx))
	ModifyStateSecurly(readonly bool, info info.IInfo, fn func (l2.State))
}

type HgHandler struct {
	State state.State
	Sigma ICore
}

func (p *HgHandler) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	for _, trx := range block.Transactions() {
		var cp chain.ChainPacket
		e := json.Unmarshal(trx, &cp)
		if e == nil {
			log.Println(string(trx))
			p.Sigma.OnChainPacket(cp)
		} else {
			log.Println(e)
		}
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

func (p *HgHandler) StateChangeHandler(state state.State) error {
	p.State = state
	return nil
}

func (p *HgHandler) SnapshotHandler(blockIndex int) ([]byte, error) {
	return []byte("statehash"), nil
}

func (p *HgHandler) RestoreHandler(snapshot []byte) ([]byte, error) {
	return []byte("statehash"), nil
}

