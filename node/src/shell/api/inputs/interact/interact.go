package inputs_interact

type InteractDto struct {
	UserId string `json:"userId" validate:"required"`
	Orig   string `json:"orig" validate:"required"`
}

func (d InteractDto) GetSpaceId() string {
	return ""
}

func (d InteractDto) GetTopicId() string {
	return ""
}

func (d InteractDto) GetMemberId() string {
	return ""
}

func (d InteractDto) Origin() string {
	return d.Orig
}
