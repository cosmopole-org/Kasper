package abstract

type IActions interface {
	Install(IState)
}

type IAction interface {
	Key() string
	Act(IState, IInput) (int, any, error)
}

type ISecureAction interface {
	Key() string
	SecurelyAct(layer ILayer, token string, packetId string, input IInput, dummy string) (int, any, error)
	SecurlyActChain(layer ILayer, token string, packetId string, input IInput, origin string)
	SecurelyActFed(layer ILayer, userId string, input IInput) (int, any, error)
}
