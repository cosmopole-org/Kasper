package inputs_machiner

type CreateInput struct {
	Username  string `json:"username" validate:"required"`
	PublicKey string `json:"publicKey"`
}

func (d CreateInput) GetData() any {
	return "dummy"
}

func (d CreateInput) GetPointId() string {
	return ""
}

func (d CreateInput) Origin() string {
	return "global"
}
