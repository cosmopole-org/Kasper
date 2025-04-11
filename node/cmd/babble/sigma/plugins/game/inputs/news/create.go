package game_inputs_news

type CreateInput struct {
	GameKey string         `json:"gameKey" validate:"required"`
	Data    map[string]any `json:"data" validate:"required"`
}

func (d CreateInput) GetSpaceId() string {
	return ""
}

func (d CreateInput) GetTopicId() string {
	return ""
}

func (d CreateInput) GetMemberId() string {
	return ""
}

func (d CreateInput) Origin() string {
	return ""
}
