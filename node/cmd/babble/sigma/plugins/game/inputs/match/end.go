package game_inputs_match

type EndInput struct {
	GameKey string   `json:"gameKey" validate:"required"`
	Level   string   `json:"level" validate:"required"`
	Winners []string `json:"winners" validate:"required"`
	Loosers []string `json:"loosers" validate:"required"`
}

func (d EndInput) GetSpaceId() string {
	return ""
}

func (d EndInput) GetTopicId() string {
	return ""
}

func (d EndInput) GetMemberId() string {
	return ""
}

func (d EndInput) Origin() string {
	return ""
}
