package model

import "kasper/src/abstract/models"

type Func func(state models.IState, input models.IInput) (any, error)

type Action struct {
	app  models.ICore
	key  string
	Func Func
}

func NewAction(app models.ICore, key string, fn Func) models.IAction {
	return &Action{app: app, key: key, Func: fn}
}

func (a *Action) App() models.ICore {
	return a.app
}

func (a *Action) Key() string {
	return a.key
}

func (a *Action) Act(state models.IState, input models.IInput) (int, any, error) {
	result, err := a.Func(state, input)
	if err != nil {
		return 0, nil, err
	}
	return 1, result, nil
}
