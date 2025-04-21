package action

import (
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/input"
	"kasper/src/abstract/state"
)

type IActions interface {
	Install(state.IState)
}

type IAction interface {
	App() core.ICore
	Key() string
	Act(state.IState, input.IInput) (int, any, error)
}

type ISecureAction interface {
	Key() string
	SecurelyAct(userId string, packetId string, packetBinary []byte, packetSignature string, input input.IInput, dummy string) (int, any, error)
	SecurlyActChain(userId string, packetId string, packetBinary []byte, packetSignature string, input input.IInput, origin string)
	SecurelyActFed(userId string, packetBinary []byte, packetSignature string, input input.IInput) (int, any, error)
}
