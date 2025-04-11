package game_inputs_match

type StartInput struct {
	GameKey string `json:"gameKey" validate:"required"`
	SpaceId string `json:"spaceId" validate:"required"`
	TopicId string `json:"topicId" validate:"required"`
}

func (d StartInput) GetSpaceId() string {
	return d.SpaceId
}

func (d StartInput) GetTopicId() string {
	return d.TopicId
}

func (d StartInput) GetMemberId() string {
	return ""
}

func (d StartInput) Origin() string {
	return ""
}
