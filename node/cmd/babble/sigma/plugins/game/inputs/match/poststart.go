package game_inputs_match

type PostStartInput struct {
	GameKey string   `json:"gameKey" validate:"required"`
	Level   string   `json:"level" validate:"required"`
	Humans  []string `json:"humans" validate:"required"`
}

func (d PostStartInput) GetSpaceId() string {
	return ""
}

func (d PostStartInput) GetTopicId() string {
	return ""
}

func (d PostStartInput) GetMemberId() string {
	return ""
}

func (d PostStartInput) Origin() string {
	return ""
}
