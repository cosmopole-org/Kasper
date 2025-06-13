package network

type IChain interface {
	Listen(port int)
	SubmitTrx(chainId string, machineId string, typ string, payload []byte)
	RegisterPipeline(pipeline func([][]byte) []string)
	NotifyNewMachineCreated(machineId string)
	Peers() []string
}
