package inputs_users

type CheckSignInput struct {
	UserId    string `json:"userId" validate:"required"`
	Payload   []byte `json:"payload" validate:"required"`
	Signature string `json:"signature" validate:"required"`
}

func (d CheckSignInput) GetData() any {
	return "dummy"
}

func (d CheckSignInput) GetPointId() string {
	return ""
}

func (d CheckSignInput) Origin() string {
	return ""
}
