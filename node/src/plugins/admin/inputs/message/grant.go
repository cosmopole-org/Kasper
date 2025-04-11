package admin_inputs_message

type GrantChatInput struct {
	UserId string `json:"userId" validate:"required"`
	Time   int64  `json:"time" validate:"required"`
}

func (d GrantChatInput) GetSpaceId() string {
	return ""
}

func (d GrantChatInput) GetTopicId() string {
	return ""
}

func (d GrantChatInput) GetMemberId() string {
	return ""
}

func (d GrantChatInput) Origin() string {
	return ""
}
