package inputs_spaces

type ReadInput struct {
	Offset int    `json:"offset"`
	Count  int    `json:"count"`
	Tag    string `json:"tag"`
	Orig   string `json:"orig"`
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
