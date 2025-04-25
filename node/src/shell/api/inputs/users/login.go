package inputs_users

type LoginInput struct {
	Username  string `json:"username" validate:"required"`
	PublicKey string `json:"publicKey" validate:"required"`
}

func (d LoginInput) GetData() any {
	return "dummy"
}

func (d LoginInput) GetPointId() string {
	return ""
}

func (d LoginInput) Origin() string {
	return ""
}
