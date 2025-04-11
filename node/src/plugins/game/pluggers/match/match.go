
	package plugger_match

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/plugins/game/actions/match"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core abstract.ICore
	}
	
		func (c *Plugger) MyLbShard() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.MyLbShard)
		}
		
		func (c *Plugger) Join() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Join)
		}
		
		func (c *Plugger) CreateInHall() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.CreateInHall)
		}
		
		func (c *Plugger) GetOpenMatches() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.GetOpenMatches)
		}
		
		func (c *Plugger) Start() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.Start)
		}
		
		func (c *Plugger) PostStart() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.PostStart)
		}
		
		func (c *Plugger) End() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.End)
		}
		
	func (c *Plugger) Install(layer abstract.ILayer, a *actions.Actions) *Plugger {
		err := actions.Install(abstract.UseToolbox[*module_model.ToolboxL2](layer.Core().Get(2).Tools()).Storage(), a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, logger *module_logger.Logger, core abstract.ICore) *Plugger {
		id := "match"
		return &Plugger{Id: &id, Actions: actions, Core: core, Logger: logger}
	}
	