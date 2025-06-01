
	package plugger_machine

	import (
		"kasper/src/abstract/models/core"
		"kasper/src/shell/utils"
	    iaction "kasper/src/abstract/models/action"
		actions "kasper/src/shell/machiner/actions/machine"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Core core.ICore
	}
	
		func (c *Plugger) Create() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Create)
		}
		
		func (c *Plugger) Deploy() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Deploy)
		}
		
	func (c *Plugger) Install(a *actions.Actions) *Plugger {
		err := actions.Install(a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, core core.ICore) *Plugger {
		id := "machine"
		return &Plugger{Id: &id, Actions: actions, Core: core}
	}
	