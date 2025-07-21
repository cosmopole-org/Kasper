package network

type INetwork interface {
	Chain() IChain
	Federation() IFederation
	Tcp() ITcp
	Ws() IWs
	Run(ports map[string]int)
}
