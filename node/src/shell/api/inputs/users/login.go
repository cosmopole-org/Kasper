package inputs_users

type LoginInput struct {
	Username  string `json:"username" validate:"required"`
}

func (d LoginInput) GetData() any {
	return "dummy"
}

func (d LoginInput) GetSpaceId() string {
	return ""
}

func (d LoginInput) GetTopicId() string {
	return ""
}

func (d LoginInput) GetMemberId() string {
	return ""
}

func (d LoginInput) Origin() string {
	return ""
}
