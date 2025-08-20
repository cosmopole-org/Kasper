
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
		
		func (c *Plugger) UploadUserEntity() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.UploadUserEntity)
		}
		
		func (c *Plugger) DeleteUserEntity() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.DeleteUserEntity)
		}
		
		func (c *Plugger) UploadPointEntity() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.UploadPointEntity)
		}
		
		func (c *Plugger) DeletePointEntity() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.DeletePointEntity)
		}
		
		func (c *Plugger) DownloadUserEntity() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.DownloadUserEntity)
		}
		
		func (c *Plugger) DownloadPointEntity() iaction.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.DownloadPointEntity)
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
	