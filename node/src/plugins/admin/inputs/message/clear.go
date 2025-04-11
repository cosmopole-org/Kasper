package admin_inputs_message

type ClearMessagesInput struct {
	TopicId string `json:"topicId" validate:"required"`
}

func (d ClearMessagesInput) GetSpaceId() string {
	return ""
}

func (d ClearMessagesInput) GetTopicId() string {
	return d.TopicId
}

func (d ClearMessagesInput) GetMemberId() string {
	return ""
}

func (d ClearMessagesInput) Origin() string {
	return ""
}
