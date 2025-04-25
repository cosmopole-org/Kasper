package plugger_auth

import (
	"kasper/src/abstract/models/action"
	"kasper/src/abstract/models/core"
	module_logger "kasper/src/core/module/logger"
	actions "kasper/src/shell/api/actions/auth"
	"kasper/src/shell/utils"
)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core core.ICore
	}
	
		func (c *Plugger) GetServerPublicKey() action.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.GetServerPublicKey)
		}
		
		func (c *Plugger) GetServersMap() action.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.GetServersMap)
		}
		
	func (c *Plugger) Install(a *actions.Actions) *Plugger {
		err := actions.Install(a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, core core.ICore) *Plugger {
		id := "auth"
		return &Plugger{Id: &id, Actions: actions, Core: core}
	}
	