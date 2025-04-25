
	package plugger_storage

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/shell/api/actions/storage"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core core.ICore
	}
	
		func (c *Plugger) Upload() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.Upload)
		}
		
		func (c *Plugger) UploadData() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.UploadData)
		}
		
		func (c *Plugger) Download() abstract.IAction {
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
	