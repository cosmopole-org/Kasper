
	package plugger_pc

	import (
		"kasper/src/abstract/models/core"
		"kasper/src/shell/utils"
	    iaction "kasper/src/abstract/models/action"
		actions "kasper/src/shell/machiner/actions/pc"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Core core.ICore
	}
	
		func (c *Plugger) RunPc() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.RunPc)
		}
		
		func (c *Plugger) ExecCommand() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.ExecCommand)
		}
		
	func (c *Plugger) Install(a *actions.Actions) *Plugger {
		err := actions.Install(a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, core core.ICore) *Plugger {
		id := "pc"
		return &Plugger{Id: &id, Actions: actions, Core: core}
	}
	