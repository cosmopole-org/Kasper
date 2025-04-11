package game_inputs_match

type GetOpenMatchesInput struct {
	GameKey string `json:"gameKey" validate:"required"`
}

func (d GetOpenMatchesInput) GetSpaceId() string {
	return ""
}

func (d GetOpenMatchesInput) GetTopicId() string {
	return ""
}

func (d GetOpenMatchesInput) GetMemberId() string {
	return ""
}

func (d GetOpenMatchesInput) Origin() string {
	return ""
}
