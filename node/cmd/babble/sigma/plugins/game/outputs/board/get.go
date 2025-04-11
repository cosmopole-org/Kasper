package game_outputs_board

import "gorm.io/datatypes"

type PreparedPlayer struct {
	UserId  string         `json:"userId"`
	Profile datatypes.JSON `json:"profile"`
	Score   datatypes.JSON `json:"score"`
}

type GetOutput struct {
	Players []PreparedPlayer `json:"players"`
}
