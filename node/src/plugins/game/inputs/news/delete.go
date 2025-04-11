package game_inputs_news

type DeleteInput struct {
	NewsId string `json:"newsId" validate:"required"`
}

func (d DeleteInput) GetSpaceId() string {
	return ""
}

func (d DeleteInput) GetTopicId() string {
	return ""
}

func (d DeleteInput) GetMemberId() string {
	return ""
}

func (d DeleteInput) Origin() string {
	return ""
}
