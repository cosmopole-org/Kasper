
	package plugger_message

	import (
		"kasper/src/abstract"
		"kasper/src/shell/utils"
		module_logger "kasper/src/core/module/logger"
		actions "kasper/src/plugins/social/actions/message"
		"kasper/src/shell/layer2/model"
	)
	
	type Plugger struct {
		Id      *string
		Actions *actions.Actions
		Logger *module_logger.Logger
		Core core.ICore
	}
	
		func (c *Plugger) CreateMessage() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.CreateMessage)
		}
		
		func (c *Plugger) UpdateMessage() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.UpdateMessage)
		}
		
		func (c *Plugger) DeleteMessage() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.DeleteMessage)
		}
		
		func (c *Plugger) SeeChat() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.SeeChat)
		}
		
		func (c *Plugger) ReadChats() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.ReadChats)
		}
		
		func (c *Plugger) ReadMessages() abstract.IAction {
			return utils.ExtractSecureAction(c.Core, c.Actions.ReadMessages)
		}
		
	func (c *Plugger) Install(a *actions.Actions) *Plugger {
		err := actions.Install(a)
		if err != nil {
			panic(err)
		}
		return c
	}

	func New(actions *actions.Actions, core core.ICore) *Plugger {
		id := "message"
		return &Plugger{Id: &id, Actions: actions, Core: core}
	}
	