package inputs_spaces

import "kasper/src/shell/utils/origin"

type UpdateMemberInput struct {
	MemberId string `json:"memberId" validate:"required"`
	SpaceId  string `json:"spaceId" validate:"required"`
	TopicId  string `json:"topicId"`
	Metadata string `json:"metadata" validate:"required"`
}

func (d UpdateMemberInput) GetData() any {
	return "dummy"
}

func (d UpdateMemberInput) GetSpaceId() string {
	return d.SpaceId
}

func (d UpdateMemberInput) GetTopicId() string {
	return d.TopicId
}

func (d UpdateMemberInput) GetMemberId() string {
	return ""
}

func (d UpdateMemberInput) Origin() string {
	return origin.FindOrigin(d.SpaceId)
}
