package network

type IChain interface {
	Run(port int)
	SubmitTrx(typ string, payload []byte)
	RegisterPipeline(pipeline func([][]byte))
	Peers() []string
}
