package network

type INetwork interface {
	Federation() IFederation
	Tcp() ITcp
}
