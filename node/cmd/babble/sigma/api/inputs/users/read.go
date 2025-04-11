package inputs_users

type ReadInput struct {
	Typ  string `json:"type" validate:"required"`
	Orig string `json:"orig"`
}

func (d ReadInput) GetData() any {
	return "dummy"
}

func (d ReadInput) GetSpaceId() string {
	return ""
}

func (d ReadInput) GetTopicId() string {
	return ""
}

func (d ReadInput) GetMemberId() string {
	return ""
}

func (d ReadInput) Origin() string {
	return d.Orig
}
