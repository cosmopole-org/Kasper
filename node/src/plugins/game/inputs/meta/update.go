package game_inputs_meta

type UpdateInput struct {
	GameKey string         `json:"gameKey" validate:"required"`
	Data    map[string]any `json:"data" validate:"required"`
	Tag     string         `json:"tag"`
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
