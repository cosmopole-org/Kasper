package actions_chain

import (
	"errors"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/state"
	inputs_chain "kasper/src/shell/api/inputs/chain"
	"kasper/src/shell/api/model"
)

type Actions struct {
	App core.ICore
}

func Install(a *Actions) error {
	return nil
}

// Create /chains/create check [ true false false ] access [ true false false false POST ]
func (a *Actions) Create(state state.IState, input inputs_chain.CreateInput) (any, error) {
	for orig, stake := range input.Participants {
		userId := a.App.Tools().Network().Chain().GetNodeOwnerId(orig)
		if userId == "" {
			return nil, errors.New("node owner not identified")
		}
		user := model.User{Id: userId}.Pull(state.Trx())
		if user.Balance < stake {
			return nil, errors.New("node owner balance is not enough")
		}
	}
	id := int64(0)
	if *input.IsTemp {
		id = a.App.Tools().Network().Chain().CreateTempChain(input.Participants)
	} else {
		id = a.App.Tools().Network().Chain().CreateWorkChain(input.Participants)
	}
	return map[string]any{"chainId": id}, nil
}
