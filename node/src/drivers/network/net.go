package tool_net

import (
	"kasper/src/abstract/adapters/network"
	"kasper/src/abstract/adapters/security"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/core"
	"kasper/src/drivers/network/tcp"
)

type Network struct {
	core core.ICore
	tcp  network.ITcp
	fed  network.IFederation
}

func (n *Network) Tcp() network.ITcp {
	return n.tcp
}

func (n *Network) Federation() network.IFederation {
	return n.fed
}

func NewNetwork(
	core core.ICore,
	storage storage.IStorage,
	security security.ISecurity,
	signaler signaler.ISignaler) *Network {
	net := &Network{
		core: core,
		tcp: tcp.NewTcp(core),
	}
	return net
}

func (net *Network) Run(ports map[string]int) {
	tcpPort, ok := ports["tcp"]
	if ok {
		net.tcp.Listen(tcpPort)
	}
	net.core.Run()
}
