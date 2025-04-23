package action

import (
	"kasper/src/abstract/models/input"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/state"
)

type IActions interface {
	Install(state.IState)
}

type IAction interface {
	StateModifier() func(bool, func(trx.ITrx))
	Key() string
	Act(state.IState, input.IInput) (int, any, error)
}

type ISecureAction interface {
	Key() string
	HasGlobalParser() bool
	ParseInput(protocol string, raw interface{}) (input.IInput, []byte, string, error)
	SecurelyAct(userId string, packetId string, packetBinary []byte, packetSignature string, input input.IInput, dummy string) (int, any, error)
	SecurlyActChain(userId string, packetId string, packetBinary []byte, packetSignature string, input input.IInput, origin string)
	SecurelyActFed(userId string, packetBinary []byte, packetSignature string, input input.IInput) (int, any, error)
}
