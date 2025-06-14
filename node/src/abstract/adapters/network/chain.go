package network

type IChain interface {
	Listen(port int)
	SubmitTrx(chainId string, machineId string, typ string, payload []byte)
	RegisterPipeline(pipeline func([][]byte) []string)
	NotifyNewMachineCreated(chainId int64, machineId string)
	CreateTempChain(participants []string) int64
	CreateWorkChain(firstNodeOrigin string) int64
	Peers() []string
}
