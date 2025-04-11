package game_inputs_board

type RankInput struct {
	Level   string `json:"level" validate:"required"`
	GameKey string `json:"gameKey" validate:"required"`
}

func (d RankInput) GetSpaceId() string {
	return ""
}

func (d RankInput) GetTopicId() string {
	return ""
}

func (d RankInput) GetMemberId() string {
	return ""
}

func (d RankInput) Origin() string {
	return ""
}
