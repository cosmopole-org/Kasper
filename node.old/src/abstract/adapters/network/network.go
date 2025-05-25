package network

type INetwork interface {
	Chain() IChain
	Federation() IFederation
	Tcp() ITcp
	Run(ports map[string]int)
}
