
		package plugger_game

		import (
			"reflect"
			"kasper/src/abstract"
			module_logger "kasper/src/core/module/logger"

		
			plugger_auth "kasper/src/plugins/game/pluggers/auth"
			action_auth "kasper/src/plugins/game/actions/auth"
			
			plugger_board "kasper/src/plugins/game/pluggers/board"
			action_board "kasper/src/plugins/game/actions/board"
			
			plugger_match "kasper/src/plugins/game/pluggers/match"
			action_match "kasper/src/plugins/game/actions/match"
			
			plugger_meta "kasper/src/plugins/game/pluggers/meta"
			action_meta "kasper/src/plugins/game/actions/meta"
			
			plugger_news "kasper/src/plugins/game/pluggers/news"
			action_news "kasper/src/plugins/game/actions/news"
			
			plugger_player "kasper/src/plugins/game/pluggers/player"
			action_player "kasper/src/plugins/game/actions/player"
			
			plugger_random "kasper/src/plugins/game/pluggers/random"
			action_random "kasper/src/plugins/game/actions/random"
			
			plugger_store "kasper/src/plugins/game/pluggers/store"
			action_store "kasper/src/plugins/game/actions/store"
			
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
			
				a_match := &action_match.Actions{Layer: layer}
				p_match := plugger_match.New(a_match, logger, core)
				PlugThePlugger(layer, p_match)
				p_match.Install(layer, a_match)
			
				a_meta := &action_meta.Actions{Layer: layer}
				p_meta := plugger_meta.New(a_meta, logger, core)
				PlugThePlugger(layer, p_meta)
				p_meta.Install(layer, a_meta)
			
				a_news := &action_news.Actions{Layer: layer}
				p_news := plugger_news.New(a_news, logger, core)
				PlugThePlugger(layer, p_news)
				p_news.Install(layer, a_news)
			
				a_player := &action_player.Actions{Layer: layer}
				p_player := plugger_player.New(a_player, logger, core)
				PlugThePlugger(layer, p_player)
				p_player.Install(layer, a_player)
			
				a_random := &action_random.Actions{Layer: layer}
				p_random := plugger_random.New(a_random, logger, core)
				PlugThePlugger(layer, p_random)
				p_random.Install(layer, a_random)
			
				a_store := &action_store.Actions{Layer: layer}
				p_store := plugger_store.New(a_store, logger, core)
				PlugThePlugger(layer, p_store)
				p_store.Install(layer, a_store)
			
		}
		