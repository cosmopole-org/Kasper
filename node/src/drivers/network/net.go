package tool_net

import (
	"crypto/tls"
	"kasper/src/abstract/adapters/network"
	"kasper/src/abstract/adapters/security"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/core"
	"kasper/src/drivers/network/chain"
	"kasper/src/drivers/network/tcp"
	"net/http"
    "golang.org/x/crypto/acme/autocert"
)

type Network struct {
	core  core.ICore
	tcp   network.ITcp
	fed   network.IFederation
	chain network.IChain
}

func (n *Network) Tcp() network.ITcp {
	return n.tcp
}

func (n *Network) Federation() network.IFederation {
	return n.fed
}

func (n *Network) Chain() network.IChain {
	return n.chain
}

func NewNetwork(
	core core.ICore,
	storage storage.IStorage,
	security security.ISecurity,
	signaler signaler.ISignaler,
	fed network.IFederation) *Network {
	net := &Network{
		core: core,
		tcp:  tcp.NewTcp(core),
		fed:  fed,
		chain: chain.NewChain(core),
	}
	return net
}

func (net *Network) Run(ports map[string]int) {
	manager := autocert.Manager{
        Cache:      autocert.DirCache("certs"),
        Prompt:     autocert.AcceptTOS,
        HostPolicy: autocert.HostWhitelist("api.decillionai.com"),
    }

    config := &tls.Config{
        GetCertificate: manager.GetCertificate,
    }

    // This starts HTTP server on :80 for challenges
    go func() {
        http.ListenAndServe(":80", manager.HTTPHandler(nil))
    }()

	tcpPort, ok := ports["tcp"]
	if ok {
		net.tcp.Listen(tcpPort, config)
	}
	net.fed.Listen(ports["fed"], config)
	net.chain.Listen(ports["chain"], config)
}
