
		package plugger_machiner

		import (
			"reflect"
			"kasper/src/abstract"
			module_logger "kasper/src/core/module/logger"

		
			plugger_machine "kasper/src/shell/machiner/pluggers/machine"
			action_machine "kasper/src/shell/machiner/actions/machine"
			
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
		
				a_machine := &action_machine.Actions{Layer: layer}
				p_machine := plugger_machine.New(a_machine, logger, core)
				PlugThePlugger(layer, p_machine)
				p_machine.Install(layer, a_machine)
			
		}
		