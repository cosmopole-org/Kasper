package admin_inputs_report

type ResolveReportsInput struct {
	ReportId string `json:"reportId" validate:"required"`
}

func (d ResolveReportsInput) GetSpaceId() string {
	return ""
}

func (d ResolveReportsInput) GetTopicId() string {
	return ""
}

func (d ResolveReportsInput) GetMemberId() string {
	return ""
}

func (d ResolveReportsInput) Origin() string {
	return ""
}
