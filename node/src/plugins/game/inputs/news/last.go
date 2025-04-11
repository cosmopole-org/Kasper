package game_inputs_news

type LastInput struct {
	GameKey string `json:"gameKey" validate:"required"`
}

func (d LastInput) GetSpaceId() string {
	return ""
}

func (d LastInput) GetTopicId() string {
	return ""
}

func (d LastInput) GetMemberId() string {
	return ""
}

func (d LastInput) Origin() string {
	return ""
}
