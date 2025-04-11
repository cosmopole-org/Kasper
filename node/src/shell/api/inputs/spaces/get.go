package inputs_spaces

import "kasper/src/shell/utils/origin"

type GetInput struct {
	SpaceId string `json:"spaceId" validate:"required"`
}

func (d GetInput) GetData() any {
	return "dummy"
}

func (d GetInput) GetSpaceId() string {
	return d.SpaceId
}

func (d GetInput) GetTopicId() string {
	return ""
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
