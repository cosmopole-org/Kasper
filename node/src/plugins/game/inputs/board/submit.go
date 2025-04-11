package game_inputs_board

type SubmitInput struct {
	GameKey string         `json:"gameKey" validate:"required"`
	Level   string         `json:"level" validate:"required"`
	Data    map[string]any `json:"data" validate:"required"`
}

func (d SubmitInput) GetSpaceId() string {
	return ""
}

func (d SubmitInput) GetTopicId() string {
	return ""
}

func (d SubmitInput) GetMemberId() string {
	return ""
}

func (d SubmitInput) Origin() string {
	return ""
}
