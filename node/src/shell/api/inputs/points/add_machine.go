package inputs_points

import "kasper/src/shell/utils/origin"

type AddMachineInput struct {
	AppId       string      `json:"userId" validate:"required"`
	PointId     string      `json:"pointId" validate:"required"`
	MachineMeta MachineMeta `json:"machinesMeta" validate:"required"`
}

func (d AddMachineInput) GetData() any {
	return "dummy"
}

func (d AddMachineInput) GetPointId() string {
	return d.PointId
}

func (d AddMachineInput) Origin() string {
	return origin.FindOrigin(d.PointId)
}
