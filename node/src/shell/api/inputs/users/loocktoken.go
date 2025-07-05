package inputs_users

type LockTokenInput struct {
	Amount        int64  `json:"amount" validate:"required"`
	ExecMachineId string `json:"execMachineId" validate:"required"`
}

func (d LockTokenInput) GetData() any {
	return "dummy"
}

func (d LockTokenInput) GetPointId() string {
	return ""
}

func (d LockTokenInput) Origin() string {
	return "global"
}
