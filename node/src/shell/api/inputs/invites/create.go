package inputs_invites

import "kasper/src/shell/utils/origin"

type CreateInput struct {
	UserId  string `json:"userId" validate:"required"`
	SpaceId string `json:"spaceId" validate:"required"`
}

func (d CreateInput) GetData() any {
	return "dummy"
}

func (d CreateInput) GetSpaceId() string {
	return d.SpaceId
}

func (d CreateInput) GetTopicId() string {
	return ""
}

func (d CreateInput) GetMemberId() string {
	return ""
}

func (d CreateInput) Origin() string {
	return origin.FindOrigin(d.SpaceId)
}
