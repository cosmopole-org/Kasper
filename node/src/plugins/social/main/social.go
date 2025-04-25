
		package plugger_social

		import (
			"reflect"
			"kasper/src/abstract"
			module_logger "kasper/src/core/module/logger"

		
			plugger_message "kasper/src/plugins/social/pluggers/message"
			action_message "kasper/src/plugins/social/actions/message"
			
			plugger_report "kasper/src/plugins/social/pluggers/report"
			action_report "kasper/src/plugins/social/actions/report"
			
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
		
				a_message := &action_message.Actions{App: core}
				p_message := plugger_message.New(a_message, core)
				PlugThePlugger(p_message)
				p_message.Install(a_message)
			
				a_report := &action_report.Actions{App: core}
				p_report := plugger_report.New(a_report, core)
				PlugThePlugger(p_report)
				p_report.Install(a_report)
			
		}
		