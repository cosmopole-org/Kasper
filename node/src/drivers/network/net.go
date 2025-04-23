package tool_net

import (
	"kasper/src/abstract/adapters/network"
	"kasper/src/abstract/adapters/security"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/core"
	modulelogger "kasper/src/core/module/logger"
	netgrpc "kasper/src/drivers/network/grpc"
	nethttp "kasper/src/drivers/network/http"
	netws "kasper/src/drivers/network/ws"
)

type Network struct {
	core core.ICore
	Http *nethttp.HttpServer
	Grpc *netgrpc.GrpcServer
	Ws   *netws.WsServer
	Fed  network.IFederation
}

func NewNetwork(
	core core.ICore,
	logger *modulelogger.Logger,
	storage storage.IStorage,
	security security.ISecurity,
	signaler signaler.ISignaler) *Network {
	hs := nethttp.New(core, logger, 0)
	net := &Network{
		core: core,
		Http: hs,
		Ws:   netws.New(core.Actor(), hs, security, signaler, storage),
		Grpc: netgrpc.New(core, logger),
	}
	return net
}

func (net *Network) Run(ports map[string]int) {
	httpPort, ok := ports["http"]
	if ok {
		net.Http.Listen(httpPort)
	}
	grpcPort, ok3 := ports["grpc"]
	if ok3 {
		net.Grpc.Listen(grpcPort)
	}
	net.core.Run()
}
