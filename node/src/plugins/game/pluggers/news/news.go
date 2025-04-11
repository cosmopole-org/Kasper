
	package plugger_news

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/plugins/game/actions/news"
		"kasper/src/shell/layer2/model"
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
		
		func (c *Plugger) Delete() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Delete)
		}
		
		func (c *Plugger) Read() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Read)
		}
		
		func (c *Plugger) Last() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Last)
		}
		
		func (c *Plugger) See() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.See)
		}
		
	func (c *Plugger) Install(layer abstract.ILayer, a *actions.Actions) *Plugger {
		err := actions.Install(abstract.UseToolbox[*module_model.ToolboxL2](layer.Core().Get(2).Tools()).Storage(), a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, logger *module_logger.Logger, core abstract.ICore) *Plugger {
		id := "news"
		return &Plugger{Id: &id, Actions: actions, Core: core, Logger: logger}
	}
	