package actions_chain

import (
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/state"
	inputs_chain "kasper/src/shell/api/inputs/chain"
)

type Actions struct {
	App core.ICore
}

func Install(a *Actions) error {
	return nil
}

// Create /chains/create check [ true false false ] access [ true false false false POST ]
func (a *Actions) Create(state state.IState, input inputs_chain.CreateInput) (any, error) {
	id := int64(0)
	if *input.IsTemp {
		id = a.App.Tools().Network().Chain().CreateTempChain(input.Participants)
	} else {
		id = a.App.Tools().Network().Chain().CreateWorkChain(input.Participants[0])
	}
	return map[string]any{"chainId": id}, nil
}
