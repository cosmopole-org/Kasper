package abstract

import (
	"encoding/json"
	"kasper/src/babble"
	"kasper/src/hashgraph"
	"kasper/src/node/state"
	"kasper/src/proxy"
	"log"
)

type Counter struct {
	Id    string `json:"id" gorm:"primaryKey;column:id"`
	Value int64  `json:"value" gorm:"column:value"`
}

type ICore interface {
	Id() string
	Gods() []string
	Layers() []ILayer
	Load([]string, []ILayer, []interface{})
	Push(ILayer)
	Get(int) ILayer
	Utils() IUtils
	ExecAppletRequestOnChain(machineId string, key string, packet []byte, userId string, callback func([]byte, int, error))
	ExecBaseRequestOnChain(key string, packet any, layer int, token string, callback func([]byte, int, error))
	ExecAppletResponseOnChain(callbackId string, packet []byte, resCode int, e string, updates []Update, cacheUpdates []CacheUpdate)
	ExecBaseResponseOnChain(callbackId string, packet any, resCode int, e string, updates []Update, cacheUpdates []CacheUpdate)
	OnChainPacket(packet ChainPacket)
	AppPendingTrxs()
	Chain() *babble.Babble
	Run()
	NewHgHandler() *HgHandler
	IpAddr() string
}

type HgHandler struct {
	State state.State
	Sigma ICore
}

func (p *HgHandler) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	for _, trx := range block.Transactions() {
		var cp ChainPacket
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

