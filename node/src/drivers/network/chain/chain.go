package chain

import (
	"crypto/tls"
	"kasper/src/abstract/models/core"
	"kasper/src/drivers/network/chain/babble"
	"kasper/src/drivers/network/chain/config"
	"kasper/src/drivers/network/chain/hashgraph"
	"kasper/src/drivers/network/chain/node/state"
	"kasper/src/drivers/network/chain/proxy"
	"kasper/src/drivers/network/chain/proxy/inmem"
	"kasper/src/shell/utils/future"
	"os"
	"strings"
)

type Blockchain struct {
	app        core.ICore
	babbleInst *babble.Babble
	proxy      *inmem.InmemProxy
	pipeline   func([][]byte) []string
}

type Ok struct {
	Proof  string
	RndNum int
}

type CLIConfig struct {
	Babble     config.Config `mapstructure:",squash"`
	ProxyAddr  string        `mapstructure:"proxy-listen"`
	ClientAddr string        `mapstructure:"client-connect"`
}

func NewChain(core core.ICore) *Blockchain {
	blockchain := &Blockchain{
		app: core,
	}
	_config := &CLIConfig{
		Babble:     *config.NewDefaultConfig(os.Getenv("IPADDR") + ":" + os.Getenv("BLOCKCHAIN_API_PORT")),
		ProxyAddr:  "127.0.0.1:1338",
		ClientAddr: "127.0.0.1:1339",
	}
	handler := &HgHandler{
		Chain: blockchain,
	}
	proxy := inmem.NewInmemProxy(handler, nil)
	_config.Babble.Proxy = proxy
	engine := babble.NewBabble(&_config.Babble)
	if err := engine.Init(); err != nil {
		_config.Babble.Logger().Error("Cannot initialize engine:", err)
		panic(err)
	}
	blockchain.babbleInst = engine
	blockchain.proxy = proxy
	return blockchain
}

func (b *Blockchain) Listen(port int, tlsConfig *tls.Config) {
	future.Async(func() {
		b.babbleInst.Run()
	}, false)
}

func (b *Blockchain) Close() {
	b.babbleInst.Node.Leave()
}

func (c *Blockchain) RegisterPipeline(pipeline func([][]byte) []string) {
	c.pipeline = pipeline
}

func (c *Blockchain) Peers() []string {
	peers := []string{}
	for _, peer := range c.babbleInst.Peers.Peers {
		peers = append(peers, strings.Split(peer.NetAddr, ":")[0])
	}
	return peers
}

func (c *Blockchain) SubmitTrx(chainId string, machineId string, typ string, payload []byte) {
	c.proxy.SubmitTx(payload)
}

func (c *Blockchain) NotifyNewMachineCreated(chainId int64, machineId string) {

}

func (c *Blockchain) CreateTempChain(peers map[string]int64) int64 {
	return 0
}

func (c *Blockchain) CreateWorkChain(peers map[string]int64) int64 {
	return 0
}

func (c *Blockchain) UserOwnsOrigin(userId string, origin string) bool {
	return true
}

func (c *Blockchain) GetNodeOwnerId(origin string) string {
	return ""
}

func (c *Blockchain) GetValidatorsOfMachineShard(machineId string) []string {
	validators := []string{}
	for k, _ := range c.app.Executors() {
		validators = append(validators, k)
	}
	return validators
}

type HgHandler struct {
	State state.State
	Chain *Blockchain
}

func (p *HgHandler) CommitHandler(block hashgraph.Block) (proxy.CommitResponse, error) {
	p.Chain.pipeline(block.Transactions())

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
