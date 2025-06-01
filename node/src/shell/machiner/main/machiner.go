
		package plugger_machiner

		import (
			"reflect"
			iaction "kasper/src/abstract/models/action"
			"kasper/src/abstract/models/core"

		
			plugger_machine "kasper/src/shell/machiner/pluggers/machine"
			action_machine "kasper/src/shell/machiner/actions/machine"
			
		)

		func PlugThePlugger(core core.ICore, plugger interface{}) {
			s := reflect.TypeOf(plugger)
			for i := 0; i < s.NumMethod(); i++ {
				f := s.Method(i)
				if f.Name != "Install" {
					result := f.Func.Call([]reflect.Value{reflect.ValueOf(plugger)})
					action := result[0].Interface().(iaction.IAction)
					core.Actor().InjectAction(action)
				}
			}
		}
	
		func PlugAll(core core.ICore) {
		
				a_machine := &action_machine.Actions{App: core}
				p_machine := plugger_machine.New(a_machine, core)
				PlugThePlugger(core, p_machine)
				p_machine.Install(a_machine)
			
		}
		