package network

type INetwork interface {
	Federation() IFederation
	Http()       IHttp
	Ws()         IWs
	Grpc()       IGrpc
}