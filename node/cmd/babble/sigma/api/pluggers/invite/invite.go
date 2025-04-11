
	package plugger_invite

	import (
		"kasper/cmd/babble/sigma/abstract"
		"kasper/cmd/babble/sigma/utils"
		module_logger "kasper/cmd/babble/sigma/core/module/logger"
		actions "kasper/cmd/babble/sigma/api/actions/invite"
		"kasper/cmd/babble/sigma/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core abstract.ICore
	}
	
		func (c *Plugger) Create() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Create)
		}
		
		func (c *Plugger) Cancel() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Cancel)
		}
		
		func (c *Plugger) Accept() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Accept)
		}
		
		func (c *Plugger) Decline() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Decline)
		}
		
	func (c *Plugger) Install(layer abstract.ILayer, a *actions.Actions) *Plugger {
		err := actions.Install(abstract.UseToolbox[*module_model.ToolboxL2](layer.Core().Get(2).Tools()).Storage(), a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, logger *module_logger.Logger, core abstract.ICore) *Plugger {
		id := "invite"
		return &Plugger{Id: &id, Actions: actions, Core: core, Logger: logger}
	}
	