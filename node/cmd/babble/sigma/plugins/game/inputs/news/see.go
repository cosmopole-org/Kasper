package game_inputs_news

type SeeInput struct {
	NewsId  string `json:"newsId" validate:"required"`
}

func (d SeeInput) GetSpaceId() string {
	return ""
}

func (d SeeInput) GetTopicId() string {
	return ""
}

func (d SeeInput) GetMemberId() string {
	return ""
}

func (d SeeInput) Origin() string {
	return ""
}
