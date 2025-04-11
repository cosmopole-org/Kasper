package game_outputs_player

type GetOutput struct {
	UserId string         `json:"userId"`
	Data   map[string]any `json:"data"`
}
