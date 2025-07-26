
	package plugger_user

	import (
		"kasper/src/abstract/models/core"
		"kasper/src/shell/utils"
	    iaction "kasper/src/abstract/models/action"
		actions "kasper/src/shell/api/actions/user"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Core core.ICore
	}
	
		func (c *Plugger) Authenticate() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Authenticate)
		}
		
		func (c *Plugger) Transfer() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Transfer)
		}
		
		func (c *Plugger) Mint() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Mint)
		}
		
		func (c *Plugger) CheckSign() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.CheckSign)
		}
		
		func (c *Plugger) LockToken() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.LockToken)
		}
		
		func (c *Plugger) ConsumeLock() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.ConsumeLock)
		}
		
		func (c *Plugger) Login() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Login)
		}
		
		func (c *Plugger) Create() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Create)
		}
		
		func (c *Plugger) Update() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Update)
		}
		
		func (c *Plugger) Get() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Get)
		}
		
		func (c *Plugger) List() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.List)
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
	