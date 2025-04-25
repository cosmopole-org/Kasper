
		package plugger_api

		import (
			"reflect"
			"kasper/src/abstract"
			module_logger "kasper/src/core/module/logger"

		
			plugger_auth "kasper/src/shell/api/pluggers/auth"
			action_auth "kasper/src/shell/api/actions/auth"
			
			plugger_dummy "kasper/src/shell/api/pluggers/dummy"
			action_dummy "kasper/src/shell/api/actions/dummy"
			
			plugger_invite "kasper/src/shell/api/pluggers/invite"
			action_invite "kasper/src/shell/api/actions/invite"
			
			plugger_point "kasper/src/shell/api/pluggers/point"
			action_point "kasper/src/shell/api/actions/point"
			
			plugger_storage "kasper/src/shell/api/pluggers/storage"
			action_storage "kasper/src/shell/api/actions/storage"
			
			plugger_user "kasper/src/shell/api/pluggers/user"
			action_user "kasper/src/shell/api/actions/user"
			
		)

		func PlugThePlugger(core core.ICore, plugger interface{}) {
			s := reflect.TypeOf(plugger)
			for i := 0; i < s.NumMethod(); i++ {
				f := s.Method(i)
				if f.Name != "Install" {
					result := f.Func.Call([]reflect.Value{reflect.ValueOf(plugger)})
					action := result[0].Interface().(abstract.IAction)
					core.Actor().InjectAction(action)
				}
			}
		}
	
		func PlugAll(core core.ICore) {
		
				a_auth := &action_auth.Actions{App: core}
				p_auth := plugger_auth.New(a_auth, core)
				PlugThePlugger(p_auth)
				p_auth.Install(a_auth)
			
				a_dummy := &action_dummy.Actions{App: core}
				p_dummy := plugger_dummy.New(a_dummy, core)
				PlugThePlugger(p_dummy)
				p_dummy.Install(a_dummy)
			
				a_invite := &action_invite.Actions{App: core}
				p_invite := plugger_invite.New(a_invite, core)
				PlugThePlugger(p_invite)
				p_invite.Install(a_invite)
			
				a_point := &action_point.Actions{App: core}
				p_point := plugger_point.New(a_point, core)
				PlugThePlugger(p_point)
				p_point.Install(a_point)
			
				a_storage := &action_storage.Actions{App: core}
				p_storage := plugger_storage.New(a_storage, core)
				PlugThePlugger(p_storage)
				p_storage.Install(a_storage)
			
				a_user := &action_user.Actions{App: core}
				p_user := plugger_user.New(a_user, core)
				PlugThePlugger(p_user)
				p_user.Install(a_user)
			
		}
		