
	package plugger_dummy

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/shell/api/actions/dummy"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core core.ICore
	}
	
		func (c *Plugger) Hello() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Hello)
		}
		
		func (c *Plugger) Time() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Time)
		}
		
		func (c *Plugger) Ping() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Ping)
		}
		
	func (c *Plugger) Install(a *actions.Actions) *Plugger {
		err := actions.Install(a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, core core.ICore) *Plugger {
		id := "dummy"
		return &Plugger{Id: &id, Actions: actions, Core: core}
	}
	