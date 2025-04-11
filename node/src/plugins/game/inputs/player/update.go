package game_inputs_player

type UpdateInput struct {
	GameKey string         `json:"gameKey" validate:"required"`
	Data    map[string]any `json:"data" validate:"required"`
}

func (d UpdateInput) GetSpaceId() string {
	return ""
}

func (d UpdateInput) GetTopicId() string {
	return ""
}

func (d UpdateInput) GetMemberId() string {
	return ""
}

func (d UpdateInput) Origin() string {
	return ""
}
