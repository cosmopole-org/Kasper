
	package plugger_invite

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/shell/api/actions/invite"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core core.ICore
	}
	
		func (c *Plugger) Create() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Create)
		}
		
		func (c *Plugger) Cancel() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Cancel)
		}
		
		func (c *Plugger) Accept() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Accept)
		}
		
		func (c *Plugger) Decline() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Decline)
		}
		
	func (c *Plugger) Install(a *actions.Actions) *Plugger {
		err := actions.Install(a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, core core.ICore) *Plugger {
		id := "invite"
		return &Plugger{Id: &id, Actions: actions, Core: core}
	}
	