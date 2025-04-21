package tool_net

import (
	"kasper/src/abstract"
	modulelogger "kasper/src/core/module/logger"
	"kasper/src/shell/layer1/adapters"
	"kasper/src/shell/layer1/tools/security"
	"kasper/src/shell/layer1/tools/signaler"
	netgrpc "kasper/src/drivers/network/grpc"
	nethttp "kasper/src/drivers/network/http"

	netws "kasper/src/drivers/network/ws"
)

type Network struct {
	core abstract.ICore
	Http *nethttp.HttpServer
	Grpc *netgrpc.GrpcServer
	Ws   *netws.WsServer
	Fed  adapters.IFederation
}

func NewNetwork(
	core abstract.ICore,
	logger *modulelogger.Logger,
	storage adapters.IStorage,
	cache adapters.ICache,
	security *security.Security,
	signaler *signaler.Signaler) *Network {
	hs := nethttp.New(core, logger, 0)
	net := &Network{
		core: core,
		Http: hs,
		Ws:   netws.New(core, hs, security, signaler, storage),
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
