package inputs_points

type SendInput struct {
	Type    string `json:"type" validate:"required"`
	Data    string `json:"data" validate:"required"`
	PointId string `json:"pointId" validate:"required"`
	UserId  string `json:"userId"`
}

func (d SendInput) GetData() any {
	return "dummy"
}

func (d SendInput) GetPointId() string {
	return d.PointId
}

func (d SendInput) Origin() string {
	return ""
}
