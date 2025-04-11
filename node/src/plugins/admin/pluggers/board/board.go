
	package plugger_board

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/plugins/admin/actions/board"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core abstract.ICore
	}
	
		func (c *Plugger) Kickout() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Kickout)
		}
		
		func (c *Plugger) SetFormula() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.SetFormula)
		}
		
		func (c *Plugger) GetFormula() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.GetFormula)
		}
		
		func (c *Plugger) ReadFormulas() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.ReadFormulas)
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
	