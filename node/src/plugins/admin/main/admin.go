
		package plugger_admin

		import (
			"reflect"
			"kasper/src/abstract"
			module_logger "kasper/src/core/module/logger"

		
			plugger_auth "kasper/src/plugins/admin/pluggers/auth"
			action_auth "kasper/src/plugins/admin/actions/auth"
			
			plugger_board "kasper/src/plugins/admin/pluggers/board"
			action_board "kasper/src/plugins/admin/actions/board"
			
			plugger_message "kasper/src/plugins/admin/pluggers/message"
			action_message "kasper/src/plugins/admin/actions/message"
			
			plugger_meta "kasper/src/plugins/admin/pluggers/meta"
			action_meta "kasper/src/plugins/admin/actions/meta"
			
			plugger_player "kasper/src/plugins/admin/pluggers/player"
			action_player "kasper/src/plugins/admin/actions/player"
			
			plugger_report "kasper/src/plugins/admin/pluggers/report"
			action_report "kasper/src/plugins/admin/actions/report"
			
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
			
				a_board := &action_board.Actions{Layer: layer}
				p_board := plugger_board.New(a_board, logger, core)
				PlugThePlugger(layer, p_board)
				p_board.Install(layer, a_board)
			
				a_message := &action_message.Actions{Layer: layer}
				p_message := plugger_message.New(a_message, logger, core)
				PlugThePlugger(layer, p_message)
				p_message.Install(layer, a_message)
			
				a_meta := &action_meta.Actions{Layer: layer}
				p_meta := plugger_meta.New(a_meta, logger, core)
				PlugThePlugger(layer, p_meta)
				p_meta.Install(layer, a_meta)
			
				a_player := &action_player.Actions{Layer: layer}
				p_player := plugger_player.New(a_player, logger, core)
				PlugThePlugger(layer, p_player)
				p_player.Install(layer, a_player)
			
				a_report := &action_report.Actions{Layer: layer}
				p_report := plugger_report.New(a_report, logger, core)
				PlugThePlugger(layer, p_report)
				p_report.Install(layer, a_report)
			
		}
		