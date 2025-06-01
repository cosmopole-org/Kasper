package inputs_points

type HistoryInput struct {
	PointId string `json:"pointId" validate:"required"`
}

func (d HistoryInput) GetData() any {
	return "dummy"
}

func (d HistoryInput) GetPointId() string {
	return d.PointId
}

func (d HistoryInput) Origin() string {
	return ""
}
