
	package plugger_machine

	import (
		"kasper/src/abstract/models/core"
		"kasper/src/shell/utils"
	    iaction "kasper/src/abstract/models/action"
		actions "kasper/src/shell/api/actions/machine"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Core core.ICore
	}
	
		func (c *Plugger) CreateApp() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.CreateApp)
		}
		
		func (c *Plugger) CreateMachine() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.CreateMachine)
		}
		
		func (c *Plugger) Deploy() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Deploy)
		}
		
		func (c *Plugger) ListApps() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.ListApps)
		}
		
		func (c *Plugger) ListMachs() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.ListMachs)
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
	