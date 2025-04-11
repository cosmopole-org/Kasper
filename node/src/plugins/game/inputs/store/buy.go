package game_inputs_store

type BuyInput struct {
	Market        string `json:"market"`
	GameKey       string `json:"gameKey" validate:"required"`
	Product       string `json:"product" validate:"required"`
	PurchaseToken string `json:"purchaseToken" validate:"required"`
}

func (d BuyInput) GetSpaceId() string {
	return ""
}

func (d BuyInput) GetTopicId() string {
	return ""
}

func (d BuyInput) GetMemberId() string {
	return ""
}

func (d BuyInput) Origin() string {
	return ""
}
