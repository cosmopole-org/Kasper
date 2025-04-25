
	package plugger_storage

	import (
		"kasper/src/abstract/models/core"
		"kasper/src/shell/utils"
	    iaction "kasper/src/abstract/models/action"
		actions "kasper/src/shell/api/actions/storage"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Core core.ICore
	}
	
		func (c *Plugger) Upload() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Upload)
		}
		
		func (c *Plugger) UploadData() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.UploadData)
		}
		
		func (c *Plugger) Download() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Download)
		}
		
	func (c *Plugger) Install(a *actions.Actions) *Plugger {
		err := actions.Install(a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, core core.ICore) *Plugger {
		id := "storage"
		return &Plugger{Id: &id, Actions: actions, Core: core}
	}
	