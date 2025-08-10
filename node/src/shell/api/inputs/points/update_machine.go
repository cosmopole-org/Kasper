package inputs_points

import "kasper/src/shell/utils/origin"

type UpdateMachineInput struct {
	PointId    string         `json:"pointId" validate:"required"`
	AppId      string         `json:"userId" validate:"required"`
	MachineId  string         `json:"machineId" validate:"required"`
	Identifier string         `json:"identifier" validate:"required"`
	Metadata   map[string]any `json:"metadata" validate:"required"`
}

func (d UpdateMachineInput) GetData() any {
	return "dummy"
}

func (d UpdateMachineInput) GetPointId() string {
	return d.PointId
}

func (d UpdateMachineInput) Origin() string {
	return origin.FindOrigin(d.PointId)
}
