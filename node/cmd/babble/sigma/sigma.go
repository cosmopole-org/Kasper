package sigma

import (
	"kasper/cmd/babble/sigma/abstract"
	modulecore "kasper/cmd/babble/sigma/core/module/core"
)

type Sigma abstract.ICore

func NewApp(config Config) Sigma {
	return modulecore.NewCore(config.Id, config.Log)
}
