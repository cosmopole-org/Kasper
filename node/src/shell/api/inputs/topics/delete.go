package inputs_topics

import "kasper/src/shell/utils/origin"

type DeleteInput struct {
	TopicId string `json:"topicId" validate:"required"`
	SpaceId string `json:"spaceId" validate:"required"`
}

func (d DeleteInput) GetData() any {
	return "dummy"
}

func (d DeleteInput) GetSpaceId() string {
	return d.SpaceId
}

func (d DeleteInput) GetTopicId() string {
	return d.TopicId
}

func (d DeleteInput) GetMemberId() string {
	return ""
}

func (d DeleteInput) Origin() string {
	return origin.FindOrigin(d.SpaceId)
}
