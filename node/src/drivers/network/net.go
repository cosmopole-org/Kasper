package tool_net

import (
	"kasper/src/abstract/adapters/network"
	"kasper/src/abstract/adapters/security"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/core"
	modulelogger "kasper/src/core/module/logger"
	"kasper/src/drivers/network/tcp"
)

type Network struct {
	core core.ICore
	Tcp  network.ITcp
	Fed  network.IFederation
}

func NewNetwork(
	core core.ICore,
	logger *modulelogger.Logger,
	storage storage.IStorage,
	security security.ISecurity,
	signaler signaler.ISignaler) *Network {
	net := &Network{
		core: core,
		Tcp: tcp.NewTcp(core),
	}
	return net
}

func (net *Network) Run(ports map[string]int) {
	tcpPort, ok := ports["tcp"]
	if ok {
		net.Tcp.Listen(tcpPort)
	}
	net.core.Run()
}
