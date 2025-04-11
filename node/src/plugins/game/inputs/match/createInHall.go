package game_inputs_match

type CreateInHallInput struct {
	GameKey string `json:"gameKey" validate:"required"`
	Level   string `json:"level" validate:"required"`
}

func (d CreateInHallInput) GetSpaceId() string {
	return ""
}

func (d CreateInHallInput) GetTopicId() string {
	return ""
}

func (d CreateInHallInput) GetMemberId() string {
	return ""
}

func (d CreateInHallInput) Origin() string {
	return ""
}
