package game_inputs_board

type WinnerInput struct {
	GameKey string `json:"gameKey" validate:"required"`
	Level   string `json:"level" validate:"required"`
}

func (d WinnerInput) GetSpaceId() string {
	return ""
}

func (d WinnerInput) GetTopicId() string {
	return ""
}

func (d WinnerInput) GetMemberId() string {
	return ""
}

func (d WinnerInput) Origin() string {
	return ""
}
