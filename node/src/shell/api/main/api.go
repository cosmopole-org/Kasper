
		package plugger_api

		import (
			"reflect"
			"kasper/src/abstract"
			module_logger "kasper/src/core/module/logger"

		
			plugger_auth "kasper/src/shell/api/pluggers/auth"
			action_auth "kasper/src/shell/api/actions/auth"
			
			plugger_dummy "kasper/src/shell/api/pluggers/dummy"
			action_dummy "kasper/src/shell/api/actions/dummy"
			
			plugger_interact "kasper/src/shell/api/pluggers/interact"
			action_interact "kasper/src/shell/api/actions/interact"
			
			plugger_invite "kasper/src/shell/api/pluggers/invite"
			action_invite "kasper/src/shell/api/actions/invite"
			
			plugger_space "kasper/src/shell/api/pluggers/space"
			action_space "kasper/src/shell/api/actions/space"
			
			plugger_storage "kasper/src/shell/api/pluggers/storage"
			action_storage "kasper/src/shell/api/actions/storage"
			
			plugger_topic "kasper/src/shell/api/pluggers/topic"
			action_topic "kasper/src/shell/api/actions/topic"
			
			plugger_user "kasper/src/shell/api/pluggers/user"
			action_user "kasper/src/shell/api/actions/user"
			
		)

		func PlugThePlugger(layer abstract.ILayer, plugger interface{}) {
			s := reflect.TypeOf(plugger)
			for i := 0; i < s.NumMethod(); i++ {
				f := s.Method(i)
				if f.Name != "Install" {
					result := f.Func.Call([]reflect.Value{reflect.ValueOf(plugger)})
					action := result[0].Interface().(abstract.IAction)
					layer.Actor().InjectAction(action)
				}
			}
		}
	
		func PlugAll(layer abstract.ILayer, logger *module_logger.Logger, core abstract.ICore) {
		
				a_auth := &action_auth.Actions{Layer: layer}
				p_auth := plugger_auth.New(a_auth, logger, core)
				PlugThePlugger(layer, p_auth)
				p_auth.Install(layer, a_auth)
			
				a_dummy := &action_dummy.Actions{Layer: layer}
				p_dummy := plugger_dummy.New(a_dummy, logger, core)
				PlugThePlugger(layer, p_dummy)
				p_dummy.Install(layer, a_dummy)
			
				a_interact := &action_interact.Actions{Layer: layer}
				p_interact := plugger_interact.New(a_interact, logger, core)
				PlugThePlugger(layer, p_interact)
				p_interact.Install(layer, a_interact)
			
				a_invite := &action_invite.Actions{Layer: layer}
				p_invite := plugger_invite.New(a_invite, logger, core)
				PlugThePlugger(layer, p_invite)
				p_invite.Install(layer, a_invite)
			
				a_space := &action_space.Actions{Layer: layer}
				p_space := plugger_space.New(a_space, logger, core)
				PlugThePlugger(layer, p_space)
				p_space.Install(layer, a_space)
			
				a_storage := &action_storage.Actions{Layer: layer}
				p_storage := plugger_storage.New(a_storage, logger, core)
				PlugThePlugger(layer, p_storage)
				p_storage.Install(layer, a_storage)
			
				a_topic := &action_topic.Actions{Layer: layer}
				p_topic := plugger_topic.New(a_topic, logger, core)
				PlugThePlugger(layer, p_topic)
				p_topic.Install(layer, a_topic)
			
				a_user := &action_user.Actions{Layer: layer}
				p_user := plugger_user.New(a_user, logger, core)
				PlugThePlugger(layer, p_user)
				p_user.Install(layer, a_user)
			
		}
		