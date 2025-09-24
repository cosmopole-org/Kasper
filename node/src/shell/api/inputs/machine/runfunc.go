package inputs_machiner

type RunMachineInput struct {
	MachineId string `json:"machineId" validate:"required"`
	Offset    int    `json:"offset" validate:"required"`
	Count     int    `json:"count" validate:"required"`
}

func (d RunMachineInput) GetData() any {
	return "dummy"
}

func (d RunMachineInput) GetPointId() string {
	return ""
}

func (d RunMachineInput) Origin() string {
	return "global"
}
