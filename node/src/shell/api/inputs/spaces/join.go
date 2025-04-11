package inputs_spaces

import "kasper/src/shell/utils/origin"

type JoinInput struct {
	SpaceId string `json:"spaceId" validate:"required"`
}

func (d JoinInput) GetData() any {
	return "dummy"
}

func (d JoinInput) GetSpaceId() string {
	return d.SpaceId
}

func (d JoinInput) GetTopicId() string {
	return ""
}

func (d JoinInput) GetMemberId() string {
	return ""
}

func (d JoinInput) Origin() string {
	return origin.FindOrigin(d.SpaceId)
}
