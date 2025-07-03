package inputs_machiner

type CreateFuncInput struct {
	Username  string `json:"username" validate:"required"`
	AppId     string `json:"appId" validate:"required"`
	PublicKey string `json:"publicKey"`
}

func (d CreateFuncInput) GetData() any {
	return "dummy"
}

func (d CreateFuncInput) GetPointId() string {
	return ""
}

func (d CreateFuncInput) Origin() string {
	return "global"
}
