package network

type IChain interface {
	Listen(port int)
	SubmitTrx(chainId string, machineId string, typ string, payload []byte)
	RegisterPipeline(pipeline func([][]byte) []string)
	NotifyNewMachineCreated(chainId int64, machineId string)
	CreateTempChain(peers map[string]int64) int64
	CreateWorkChain(peers map[string]int64) int64
	Peers() []string
	UserOwnsOrigin(userId string, origin string) bool
	GetNodeOwnerId(origin string) string
	GetValidatorsOfMachineShard(machineId string) []string
}
