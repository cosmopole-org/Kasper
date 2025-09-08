package network

import "crypto/tls"

type IChain interface {
	Listen(port int, tlsConfig *tls.Config)
	SubmitTrx(chainId string, machineId string, typ string, payload []byte)
	RegisterPipeline(pipeline func([][]byte) []string)
	NotifyNewMachineCreated(chainId int64, machineId string)
	CreateTempChain() string
	CreateWorkChain() string
	Peers() []string
	UserOwnsOrigin(userId string, origin string) bool
	GetNodeOwnerId(origin string) string
	GetValidatorsOfMachineShard(machineId string) []string
	Close()
}
