package inputs_invites

import "kasper/cmd/babble/sigma/utils/origin"

type AcceptInput struct {
	InviteId string `json:"inviteId" validate:"required"`
}

func (d AcceptInput) GetData() any {
	return "dummy"
}

func (d AcceptInput) GetSpaceId() string {
	return ""
}

func (d AcceptInput) GetTopicId() string {
	return ""
}

func (d AcceptInput) GetMemberId() string {
	return ""
}

func (d AcceptInput) Origin() string {
	return origin.FindOrigin(d.InviteId)
}
