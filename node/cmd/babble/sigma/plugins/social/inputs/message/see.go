package inputs_message

type SeeChatInput struct {
	SpaceId string `json:"spaceId" validate:"required"`
	TopicId string `json:"topicId" validate:"required"`
}

func (d SeeChatInput) GetSpaceId() string {
	return d.SpaceId
}

func (d SeeChatInput) GetTopicId() string {
	return d.TopicId
}

func (d SeeChatInput) GetMemberId() string {
	return ""
}

func (d SeeChatInput) Origin() string {
	return ""
}
