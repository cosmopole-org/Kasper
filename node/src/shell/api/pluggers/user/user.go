
	package plugger_user

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/shell/api/actions/user"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core core.ICore
	}
	
		func (c *Plugger) Authenticate() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Authenticate)
		}
		
		func (c *Plugger) Login() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Login)
		}
		
		func (c *Plugger) Create() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Create)
		}
		
		func (c *Plugger) Get() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Get)
		}
		
	func (c *Plugger) Install(a *actions.Actions) *Plugger {
		err := actions.Install(a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, core core.ICore) *Plugger {
		id := "user"
		return &Plugger{Id: &id, Actions: actions, Core: core}
	}
	