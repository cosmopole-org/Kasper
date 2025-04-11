package inputs_machiner

type DeployInput struct {
	MachineId string `json:"machineId" validate:"required"`
	ByteCode  string `json:"byteCode" validate:"required"`
	Runtime   string `json:"runtime" validate:"required"`
}

func (d DeployInput) GetData() any {
	return "dummy"
}

func (d DeployInput) GetSpaceId() string {
	return ""
}

func (d DeployInput) GetTopicId() string {
	return ""
}

func (d DeployInput) GetMemberId() string {
	return ""
}

func (d DeployInput) Origin() string {
	return "global"
}
