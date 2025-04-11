package admin_inputs_meta

type GetInput struct {
	GameKey string `json:"gameKey" validate:"required"`
	Tag     string `json:"tag"`
}

func (d GetInput) GetSpaceId() string {
	return ""
}

func (d GetInput) GetTopicId() string {
	return ""
}

func (d GetInput) GetMemberId() string {
	return ""
}

func (d GetInput) Origin() string {
	return ""
}
