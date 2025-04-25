package kasper

import (
	"kasper/src/abstract/models/core"
	modulecore "kasper/src/core/module/core"
)

type Kasper core.ICore

func NewApp(config Config) Kasper {
	return modulecore.NewCore(config.Id)
}
