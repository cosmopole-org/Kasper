package inputs_machiner

type CreateMachineInput struct {
	Username  string `json:"username" validate:"required"`
	AppId     string `json:"appId" validate:"required"`
	Path      string `json:"path" validate:"required"`
	PublicKey string `json:"publicKey"`
}

func (d CreateMachineInput) GetData() any {
	return "dummy"
}

func (d CreateMachineInput) GetPointId() string {
	return ""
}

func (d CreateMachineInput) Origin() string {
	return "global"
}
