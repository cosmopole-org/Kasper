package admin_inputs_board

type SetFormulaInput struct {
	GameKey    string                        `json:"gameKey" validate:"required"`
	Level      string                        `json:"level" validate:"required"`
	Keys       []string                      `json:"keys" validate:"required"`
	Weights    []float64                     `json:"weights" validate:"required"`
	NonZero    []bool                        `json:"nonZero" validate:"required"`
	Operations []string                      `json:"operations" validate:"required"`
	TotalOp    string                        `json:"totalOp" validate:"required"`
	Rules      map[string]map[string]float64 `json:"rules" validate:"required"`
	Order      string                        `json:"order" validate:"required"`
}

func (d SetFormulaInput) GetSpaceId() string {
	return ""
}

func (d SetFormulaInput) GetTopicId() string {
	return ""
}

func (d SetFormulaInput) GetMemberId() string {
	return ""
}

func (d SetFormulaInput) Origin() string {
	return ""
}
