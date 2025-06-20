package kasper

import (
	"crypto/rsa"
	"kasper/src/abstract/models/core"
	modulecore "kasper/src/core/module/core"
)

type Kasper core.ICore

func NewApp(ownerId string, ownerPrivateKey *rsa.PrivateKey) Kasper {
	return modulecore.NewCore(ownerId, ownerPrivateKey)
}
