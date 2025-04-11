package inputs_spaces

import "kasper/src/shell/utils/origin"

type RemoveMemberInput struct {
	MemberId string `json:"memberId" validate:"required"`
	SpaceId  string `json:"spaceId" validate:"required"`
	TopicId  string `json:"topicId"`
}

func (d RemoveMemberInput) GetData() any {
	return "dummy"
}

func (d RemoveMemberInput) GetSpaceId() string {
	return d.SpaceId
}

func (d RemoveMemberInput) GetTopicId() string {
	return d.TopicId
}

func (d RemoveMemberInput) GetMemberId() string {
	return d.MemberId
}

func (d RemoveMemberInput) Origin() string {
	return origin.FindOrigin(d.SpaceId)
}
