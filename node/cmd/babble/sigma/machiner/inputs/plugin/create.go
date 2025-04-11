package inputs_machiner

type CreateInput struct {
	Username  string `json:"username" validate:"required"`
	PublicKey string `json:"publicKey"`
}

func (d CreateInput) GetData() any {
	return "dummy"
}

func (d CreateInput) GetSpaceId() string {
	return ""
}

func (d CreateInput) GetTopicId() string {
	return ""
}

func (d CreateInput) GetMemberId() string {
	return ""
}

func (d CreateInput) Origin() string {
	return "global"
}
