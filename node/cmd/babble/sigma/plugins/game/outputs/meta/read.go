package game_outputs_meta

import game_model "kasper/cmd/babble/sigma/plugins/game/model"

type ReadOutput struct{
	Data []game_model.Meta `json:"data"`
}
