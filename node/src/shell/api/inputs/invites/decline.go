package inputs_invites

import "kasper/src/shell/utils/origin"

type DeclineInput struct {
	InviteId string `json:"inviteId" validate:"required"`
}

func (d DeclineInput) GetData() any {
	return "dummy"
}

func (d DeclineInput) GetSpaceId() string {
	return ""
}

func (d DeclineInput) GetTopicId() string {
	return ""
}

func (d DeclineInput) GetMemberId() string {
	return ""
}

func (d DeclineInput) Origin() string {
	return origin.FindOrigin(d.InviteId)
}
