package inputs_points

type CreateInput struct {
	IsPublic *bool  `json:"isPublic" validate:"required"`
	PersHist *bool  `json:"persHist" validate:"required"`
	Orig     string `json:"orig"`
}

func (d CreateInput) GetData() any {
	return "dummy"
}

func (d CreateInput) GetPointId() string {
	return ""
}

func (d CreateInput) Origin() string {
	return d.Orig
}
