package game_outputs_board

import "gorm.io/datatypes"

type WinnerOutput struct {
	Rank datatypes.JSON `json:"rank"`
}
