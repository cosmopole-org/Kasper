package inputs_spaces

type CreatePrivateInput struct {
	ParticipantId string `json:"participantId" validate:"required"`
	Orig     string `json:"orig"`
}

func (d CreatePrivateInput) GetSpaceId() string {
	return ""
}

func (d CreatePrivateInput) GetTopicId() string {
	return ""
}

func (d CreatePrivateInput) GetMemberId() string {
	return ""
}

func (d CreatePrivateInput) Origin() string {
	return d.Orig
}