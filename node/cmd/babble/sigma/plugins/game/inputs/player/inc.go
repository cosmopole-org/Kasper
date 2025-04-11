package game_inputs_player

type IncInput struct {
	GameKey string `json:"gameKey" validate:"required"`
	IncKey  string `json:"incKey" validate:"required"`
}

func (d IncInput) GetSpaceId() string {
	return ""
}

func (d IncInput) GetTopicId() string {
	return ""
}

func (d IncInput) GetMemberId() string {
	return ""
}

func (d IncInput) Origin() string {
	return ""
}
