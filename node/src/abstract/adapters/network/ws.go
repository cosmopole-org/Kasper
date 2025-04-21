package network

import (
	"kasper/src/abstract/adapters/security"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
)

type IWs interface {
	Load(core core.ICore, httpServer *IHttp, security security.ISecurity, signaler signaler.ISignaler, storage storage.IStorage)
}
