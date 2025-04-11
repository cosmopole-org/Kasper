package inputs_topics

import "kasper/src/shell/utils/origin"

type ReadInput struct {
	SpaceId string `json:"spaceId" validate:"required"`
}

func (d ReadInput) GetData() any {
	return "dummy"
}

func (d ReadInput) GetSpaceId() string {
	return d.SpaceId
}

func (d ReadInput) GetTopicId() string {
	return ""
}

func (d ReadInput) GetMemberId() string {
	return ""
}

func (d ReadInput) Origin() string {
	return origin.FindOrigin(d.SpaceId)
}
