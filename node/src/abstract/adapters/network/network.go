package network

type INetwork interface {
	Federation() IFederation
	Tcp() ITcp
	Run(ports map[string]int)
}
