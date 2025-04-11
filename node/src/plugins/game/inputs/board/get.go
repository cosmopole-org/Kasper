package game_inputs_board

type GetInput struct {
	GameKey string `json:"gameKey" validate:"required"`
	Level   string `json:"level" validate:"required"`
}

func (d GetInput) GetSpaceId() string {
	return ""
}

func (d GetInput) GetTopicId() string {
	return ""
}

func (d GetInput) GetMemberId() string {
	return ""
}

func (d GetInput) Origin() string {
	return ""
}
