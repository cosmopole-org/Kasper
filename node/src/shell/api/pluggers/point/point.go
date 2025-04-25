
	package plugger_point

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/shell/api/actions/point"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core core.ICore
	}
	
		func (c *Plugger) AddMember() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.AddMember)
		}
		
		func (c *Plugger) UpdateMember() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.UpdateMember)
		}
		
		func (c *Plugger) ReadMembers() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.ReadMembers)
		}
		
		func (c *Plugger) RemoveMember() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.RemoveMember)
		}
		
		func (c *Plugger) Create() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Create)
		}
		
		func (c *Plugger) Update() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Update)
		}
		
		func (c *Plugger) Delete() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Delete)
		}
		
		func (c *Plugger) Get() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Get)
		}
		
		func (c *Plugger) Read() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Read)
		}
		
		func (c *Plugger) Join() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Join)
		}
		
		func (c *Plugger) Send() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Send)
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
	