package inputs_spaces

import "kasper/src/shell/utils/origin"

type DeleteInput struct {
	SpaceId string `json:"spaceId" validate:"required"`
}

func (d DeleteInput) GetData() any {
	return "dummy"
}

func (d DeleteInput) GetSpaceId() string {
	return d.SpaceId
}

func (d DeleteInput) GetTopicId() string {
	return ""
}

func (d DeleteInput) GetMemberId() string {
	return ""
}

func (d DeleteInput) Origin() string {
	return origin.FindOrigin(d.SpaceId)
}
