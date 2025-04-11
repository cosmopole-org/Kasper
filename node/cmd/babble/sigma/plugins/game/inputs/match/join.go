package game_inputs_match

type JoinInput struct {
	GameKey string `json:"gameKey" validate:"required"`
	Level   string `json:"level" validate:"required"`
}

func (d JoinInput) GetSpaceId() string {
	return ""
}

func (d JoinInput) GetTopicId() string {
	return ""
}

func (d JoinInput) GetMemberId() string {
	return ""
}

func (d JoinInput) Origin() string {
	return ""
}
