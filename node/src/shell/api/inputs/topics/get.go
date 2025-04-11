package inputs_topics

import "kasper/src/shell/utils/origin"

type GetInput struct {
	TopicId string `json:"topicId" validate:"required"`
	SpaceId string `json:"spaceId" validate:"required"`
}

func (d GetInput) GetData() any {
	return "dummy"
}

func (d GetInput) GetSpaceId() string {
	return d.SpaceId
}

func (d GetInput) GetTopicId() string {
	return d.TopicId
}

func (d GetInput) GetMemberId() string {
	return ""
}

func (d GetInput) Origin() string {
	o := origin.FindOrigin(d.SpaceId)
	if o == "global" {
		return ""
	}
	return o
}
