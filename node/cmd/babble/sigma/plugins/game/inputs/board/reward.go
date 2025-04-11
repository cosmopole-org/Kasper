package game_inputs_board

type RewardInput struct {
	GameKey string `json:"gameKey" validate:"required"`
	Level   string `json:"level" validate:"required"`
}

func (d RewardInput) GetSpaceId() string {
	return ""
}

func (d RewardInput) GetTopicId() string {
	return ""
}

func (d RewardInput) GetMemberId() string {
	return ""
}

func (d RewardInput) Origin() string {
	return ""
}
