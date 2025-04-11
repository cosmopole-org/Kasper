
	package plugger_message

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/plugins/admin/actions/message"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core abstract.ICore
	}
	
		func (c *Plugger) SwitchChatBanned() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.SwitchChatBanned)
		}
		
		func (c *Plugger) GrantChatPerm() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.GrantChatPerm)
		}
		
		func (c *Plugger) DeleteMessage() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.DeleteMessage)
		}
		
		func (c *Plugger) ClearMessages() abstract.IAction {
			return utils.ExtractSecureAction(c.Logger, c.Core, c.Actions.ClearMessages)
		}
		
	func (c *Plugger) Install(layer abstract.ILayer, a *actions.Actions) *Plugger {
		err := actions.Install(abstract.UseToolbox[*module_model.ToolboxL2](layer.Core().Get(2).Tools()).Storage(), a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, logger *module_logger.Logger, core abstract.ICore) *Plugger {
		id := "message"
		return &Plugger{Id: &id, Actions: actions, Core: core, Logger: logger}
	}
	