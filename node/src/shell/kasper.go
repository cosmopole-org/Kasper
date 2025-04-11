package kasper

import (
	"kasper/src/abstract"
	modulecore "kasper/src/core/module/core"
)

type Sigma abstract.ICore

func NewApp(config Config) Sigma {
	return modulecore.NewCore(config.Id, config.Log)
}
