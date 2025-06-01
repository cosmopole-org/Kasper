package inputs_users

import "kasper/src/shell/utils/origin"

type GetInput struct {
	UserId string `json:"userId" validate:"required"`
}

func (d GetInput) GetData() any {
	return "dummy"
}

func (d GetInput) GetPointId() string {
	return ""
}

func (d GetInput) Origin() string {
	o := origin.FindOrigin(d.UserId)
	if o == "global" {
		return ""
	}
	return o
}
