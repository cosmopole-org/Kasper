package inputs_topics

import "kasper/src/shell/utils/origin"

type CreateInput struct {
	Title    string `json:"title" validate:"required"`
	Avatar   string `json:"avatar" validate:"required"`
	SpaceId  string `json:"spaceId" validate:"required"`
	Metadata string `json:"metadata" validate:"required"`
}

func (d CreateInput) GetData() any {
	return "dummy"
}

func (d CreateInput) GetSpaceId() string {
	return d.SpaceId
}

func (d CreateInput) GetTopicId() string {
	return ""
}

func (d CreateInput) GetMemberId() string {
	return ""
}

func (d CreateInput) Origin() string {
	return origin.FindOrigin(d.SpaceId)
}
