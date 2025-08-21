package tool_net

import (
	"crypto/tls"
	"kasper/src/abstract/adapters/network"
	"kasper/src/abstract/adapters/security"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/core"
	"kasper/src/drivers/network/chain"
	"kasper/src/drivers/network/client/tcp"
	"kasper/src/drivers/network/client/ws"
	"net/http"

	"golang.org/x/crypto/acme/autocert"
)

type Network struct {
	core       core.ICore
	tcp        network.ITcp
	ws         network.IWs
	fed        network.IFederation
	chain      network.IChain
	tlsConfig  *tls.Config
	tlsManager *autocert.Manager
}

func (n *Network) Tcp() network.ITcp {
	return n.tcp
}

func (n *Network) Ws() network.IWs {
	return n.ws
}

func (n *Network) Federation() network.IFederation {
	return n.fed
}

func (n *Network) Chain() network.IChain {
	return n.chain
}

func (n *Network) TlsConfig() *tls.Config {
	return n.tlsConfig
}

func NewNetwork(
	core core.ICore,
	storage storage.IStorage,
	security security.ISecurity,
	signaler signaler.ISignaler,
	fed network.IFederation) *Network {
	manager := &autocert.Manager{
		Cache:      autocert.DirCache("certs"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("api.decillionai.com"),
	}
	config := &tls.Config{
		GetCertificate: manager.GetCertificate,
	}
	net := &Network{
		core:       core,
		tcp:        tcp.NewTcp(core),
		ws:         ws.NewWs(core),
		fed:        fed,
		chain:      chain.NewChain(core),
		tlsConfig:  config,
		tlsManager: manager,
	}
	return net
}

func (net *Network) Run(ports map[string]int) {
	// This starts HTTP server on :80 for challenges
	go func() {
		http.ListenAndServe(":80", net.tlsManager.HTTPHandler(nil))
	}()

	tcpPort, ok := ports["tcp"]
	if ok {
		net.tcp.Listen(tcpPort, net.tlsConfig)
	}
	wsPort, ok := ports["ws"]
	if ok {
		net.ws.Listen(wsPort, net.tlsConfig)
	}
	net.fed.Listen(ports["fed"], net.tlsConfig)
	net.chain.Listen(ports["chain"], net.tlsConfig)
}
