package inputs_points

import "kasper/src/shell/utils/origin"

type UpdateAppInput struct {
	PointId    string         `json:"pointId" validate:"required"`
	AppId      string         `json:"userId" validate:"required"`
	MachineId  string         `json:"machineId" validate:"required"`
	Identifier string         `json:"identifier" validate:"required"`
	Metadata   map[string]any `json:"metadata" validate:"required"`
}

func (d UpdateAppInput) GetData() any {
	return "dummy"
}

func (d UpdateAppInput) GetPointId() string {
	return d.PointId
}

func (d UpdateAppInput) Origin() string {
	return origin.FindOrigin(d.PointId)
}
