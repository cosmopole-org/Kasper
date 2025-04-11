
	package plugger_dummy

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/shell/api/actions/dummy"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core abstract.ICore
	}
	
		func (c *Plugger) Hello() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Hello)
		}
		
		func (c *Plugger) Time() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Time)
		}
		
		func (c *Plugger) Ping() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Ping)
		}
		
	func (c *Plugger) Install(layer abstract.ILayer, a *actions.Actions) *Plugger {
		err := actions.Install(abstract.UseToolbox[*module_model.ToolboxL2](layer.Core().Get(2).Tools()).Storage(), a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, logger *module_logger.Logger, core abstract.ICore) *Plugger {
		id := "dummy"
		return &Plugger{Id: &id, Actions: actions, Core: core, Logger: logger}
	}
	