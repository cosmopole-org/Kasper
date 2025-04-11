
	package plugger_board

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/plugins/game/actions/board"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core abstract.ICore
	}
	
		func (c *Plugger) Submit() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Submit)
		}
		
		func (c *Plugger) Get() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Get)
		}
		
		func (c *Plugger) Winner() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Winner)
		}
		
		func (c *Plugger) Reward() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Reward)
		}
		
		func (c *Plugger) Rank() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Rank)
		}
		
		func (c *Plugger) NextEnd() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.NextEnd)
		}
		
	func (c *Plugger) Install(layer abstract.ILayer, a *actions.Actions) *Plugger {
		err := actions.Install(abstract.UseToolbox[*module_model.ToolboxL2](layer.Core().Get(2).Tools()).Storage(), a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, logger *module_logger.Logger, core abstract.ICore) *Plugger {
		id := "board"
		return &Plugger{Id: &id, Actions: actions, Core: core, Logger: logger}
	}
	