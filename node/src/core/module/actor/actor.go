package module_actor

import (
	"kasper/src/abstract/models"
)

type Actor struct {
	actionMap map[string]models.IAction
}

func NewActor() *Actor {
	return &Actor{actionMap: make(map[string]models.IAction)}
}

func (a *Actor) InjectService(service interface{}) {
}

func (a *Actor) InjectAction(action models.IAction) {
	a.actionMap[action.Key()] = action
}

func (a *Actor) FetchAction(key string) models.IAction {
	return a.actionMap[key]
}
