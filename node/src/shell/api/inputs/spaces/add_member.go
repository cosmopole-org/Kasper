package inputs_spaces

import "kasper/src/shell/utils/origin"

type AddMemberInput struct {
	UserId   string `json:"userId" validate:"required"`
	SpaceId  string `json:"spaceId" validate:"required"`
	TopicId  string `json:"topicId"`
	Metadata string `json:"metadata" validate:"required"`
}

func (d AddMemberInput) GetData() any {
	return "dummy"
}

func (d AddMemberInput) GetSpaceId() string {
	return d.SpaceId
}

func (d AddMemberInput) GetTopicId() string {
	return ""
}

func (d AddMemberInput) GetMemberId() string {
	return ""
}

func (d AddMemberInput) Origin() string {
	return origin.FindOrigin(d.SpaceId)
}
