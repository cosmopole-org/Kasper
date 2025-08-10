
	package plugger_point

	import (
		"kasper/src/abstract/models/core"
		"kasper/src/shell/utils"
	    iaction "kasper/src/abstract/models/action"
		actions "kasper/src/shell/api/actions/point"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Core core.ICore
	}
	
		func (c *Plugger) AddApp() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.AddApp)
		}
		
		func (c *Plugger) UpdateApp() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.UpdateApp)
		}
		
		func (c *Plugger) RemoveApp() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.RemoveApp)
		}
		
		func (c *Plugger) AddMember() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.AddMember)
		}
		
		func (c *Plugger) UpdateMember() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.UpdateMember)
		}
		
		func (c *Plugger) ReadMembers() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.ReadMembers)
		}
		
		func (c *Plugger) RemoveMember() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.RemoveMember)
		}
		
		func (c *Plugger) Create() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Create)
		}
		
		func (c *Plugger) Update() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Update)
		}
		
		func (c *Plugger) Delete() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Delete)
		}
		
		func (c *Plugger) Meta() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Meta)
		}
		
		func (c *Plugger) Get() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Get)
		}
		
		func (c *Plugger) Read() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Read)
		}
		
		func (c *Plugger) Join() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Join)
		}
		
		func (c *Plugger) Signal() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Signal)
		}
		
		func (c *Plugger) History() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.History)
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
		id := "point"
		return &Plugger{Id: &id, Actions: actions, Core: core}
	}
	