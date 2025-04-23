package network

import (
	"kasper/src/abstract/adapters/security"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/action"
)

type IWs interface {
	Load(core action.IActor, httpServer *IHttp, security security.ISecurity, signaler signaler.ISignaler, storage storage.IStorage)
}
