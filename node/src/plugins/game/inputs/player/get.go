package game_inputs_player

type GetInput struct {
	GameKey string `json:"gameKey" validate:"required"`
	UserId  string `json:"userId"`
}

func (d GetInput) GetSpaceId() string {
	return ""
}

func (d GetInput) GetTopicId() string {
	return ""
}

func (d GetInput) GetMemberId() string {
	return ""
}

func (d GetInput) Origin() string {
	return ""
}
