package game_inputs_match

type MyLbShardInput struct {
	GameKey string `json:"gameKey" validate:"required"`
	Level   string `json:"level" validate:"required"`
}

func (d MyLbShardInput) GetSpaceId() string {
	return ""
}

func (d MyLbShardInput) GetTopicId() string {
	return ""
}

func (d MyLbShardInput) GetMemberId() string {
	return ""
}

func (d MyLbShardInput) Origin() string {
	return ""
}
