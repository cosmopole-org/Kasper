
	package plugger_auth

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/shell/api/actions/auth"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core abstract.ICore
	}
	
		func (c *Plugger) GetServerPublicKey() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.GetServerPublicKey)
		}
		
		func (c *Plugger) GetServersMap() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.GetServersMap)
		}
		
	func (c *Plugger) Install(layer abstract.ILayer, a *actions.Actions) *Plugger {
		err := actions.Install(abstract.UseToolbox[*module_model.ToolboxL2](layer.Core().Get(2).Tools()).Storage(), a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, logger *module_logger.Logger, core abstract.ICore) *Plugger {
		id := "auth"
		return &Plugger{Id: &id, Actions: actions, Core: core, Logger: logger}
	}
	