package inputs_machiner

type CreateAppInput struct {
	ChainId int64 `json:"chainId" validate:"required"`
}

func (d CreateAppInput) GetData() any {
	return "dummy"
}

func (d CreateAppInput) GetPointId() string {
	return ""
}

func (d CreateAppInput) Origin() string {
	return "global"
}
