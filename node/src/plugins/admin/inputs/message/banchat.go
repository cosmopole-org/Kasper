package admin_inputs_message

type BanChatInput struct {
	UserId  string `json:"userId" validate:"required"`
	GameKey string `json:"gameKey" validate:"required"`
	Banned  bool   `json:"banned" validate:"required"`
}

func (d BanChatInput) GetSpaceId() string {
	return ""
}

func (d BanChatInput) GetTopicId() string {
	return ""
}

func (d BanChatInput) GetMemberId() string {
	return ""
}

func (d BanChatInput) Origin() string {
	return ""
}
