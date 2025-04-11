
	package plugger_meta

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/plugins/admin/actions/meta"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core abstract.ICore
	}
	
		func (c *Plugger) Update() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Update)
		}
		
		func (c *Plugger) Get() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Get)
		}
		
		func (c *Plugger) Read() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Read)
		}
		
	func (c *Plugger) Install(layer abstract.ILayer, a *actions.Actions) *Plugger {
		err := actions.Install(abstract.UseToolbox[*module_model.ToolboxL2](layer.Core().Get(2).Tools()).Storage(), a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, logger *module_logger.Logger, core abstract.ICore) *Plugger {
		id := "meta"
		return &Plugger{Id: &id, Actions: actions, Core: core, Logger: logger}
	}
	