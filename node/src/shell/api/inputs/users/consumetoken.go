package inputs_users

type ConsumeTokenInput struct {
	Orig         string `json:"orig" validate:"required"`
	TokenOwnerId string `json:"tokenOwnerId" validate:"required"`
	TokenId      string `json:"tokenId" validate:"required"`
	Amount       int64  `json:"amount" validate:"required"`
}

func (d ConsumeTokenInput) GetData() any {
	return "dummy"
}

func (d ConsumeTokenInput) GetPointId() string {
	return ""
}

func (d ConsumeTokenInput) Origin() string {
	return "global"
}
