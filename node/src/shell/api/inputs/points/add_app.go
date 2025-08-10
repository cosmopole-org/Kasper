package inputs_points

import "kasper/src/shell/utils/origin"

type MachineMeta struct {
	MachineId  string         `json:"machineId" validate:"required"`
	Identifier string         `json:"identifier" validate:"required"`
	Metadata   map[string]any `json:"metadata" validate:"required"`
}

type AddAppInput struct {
	AppId        string        `json:"userId" validate:"required"`
	PointId      string        `json:"pointId" validate:"required"`
	MachinesMeta []MachineMeta `json:"machinesMeta" validate:"required"`
}

func (d AddAppInput) GetData() any {
	return "dummy"
}

func (d AddAppInput) GetPointId() string {
	return d.PointId
}

func (d AddAppInput) Origin() string {
	return origin.FindOrigin(d.PointId)
}
