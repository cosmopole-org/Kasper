package inputs_users

import "kasper/src/shell/utils/origin"

type GetInput struct {
	UserId string `json:"userId" validate:"required"`
}

func (d GetInput) GetData() any {
	return "dummy"
}

func (d GetInput) GetSpaceId() string {
	return ""
}

func (d GetInput) GetTopicId() string {
	return ""
}

func (d GetInput) GetMemberId() string {
	return ""
}

func (d GetInput) Origin() string {
	o := origin.FindOrigin(d.UserId)
	if o == "global" {
		return ""
	}
	return o
}
