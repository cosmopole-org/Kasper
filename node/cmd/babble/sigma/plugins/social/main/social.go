
		package plugger_social

		import (
			"reflect"
			"kasper/cmd/babble/sigma/abstract"
			module_logger "kasper/cmd/babble/sigma/core/module/logger"

		
			plugger_message "kasper/cmd/babble/sigma/plugins/social/pluggers/message"
			action_message "kasper/cmd/babble/sigma/plugins/social/actions/message"
			
			plugger_report "kasper/cmd/babble/sigma/plugins/social/pluggers/report"
			action_report "kasper/cmd/babble/sigma/plugins/social/actions/report"
			
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
		
				a_message := &action_message.Actions{Layer: layer}
				p_message := plugger_message.New(a_message, logger, core)
				PlugThePlugger(layer, p_message)
				p_message.Install(layer, a_message)
			
				a_report := &action_report.Actions{Layer: layer}
				p_report := plugger_report.New(a_report, logger, core)
				PlugThePlugger(layer, p_report)
				p_report.Install(layer, a_report)
			
		}
		