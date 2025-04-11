package admin_inputs_board

type GetFormulaInput struct {
	GameKey string `json:"gameKey" validate:"required"`
	Level   string `json:"level" validate:"required"`
}

func (d GetFormulaInput) GetSpaceId() string {
	return ""
}

func (d GetFormulaInput) GetTopicId() string {
	return ""
}

func (d GetFormulaInput) GetMemberId() string {
	return ""
}

func (d GetFormulaInput) Origin() string {
	return ""
}
