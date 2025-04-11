package game_inputs_news

type ReadInput struct {
	GameKey string `json:"gameKey" validate:"required"`
}

func (d ReadInput) GetSpaceId() string {
	return ""
}

func (d ReadInput) GetTopicId() string {
	return ""
}

func (d ReadInput) GetMemberId() string {
	return ""
}

func (d ReadInput) Origin() string {
	return ""
}
