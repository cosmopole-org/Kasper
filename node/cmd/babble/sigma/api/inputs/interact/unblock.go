package inputs_interact

type UnblockDto struct {
	UserId string `json:"userId" validate:"required"`
	Orig   string `json:"orig"`
}

func (d UnblockDto) GetSpaceId() string {
	return ""
}

func (d UnblockDto) GetTopicId() string {
	return ""
}

func (d UnblockDto) GetMemberId() string {
	return ""
}

func (d UnblockDto) Origin() string {
	return d.Orig
}