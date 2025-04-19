package inputs_storage

import (
	"kasper/src/shell/utils/origin"
)

type UploadDataInput struct {
	Data    string `json:"data" validate:"required"`
	SpaceId string `json:"spaceId" validate:"required"`
	TopicId string `json:"topicId" validate:"required"`
	FileId  string `json:"fileId"`
}

func (d UploadDataInput) GetData() any {
	return "dummy"
}

func (d UploadDataInput) GetSpaceId() string {
	return d.SpaceId
}

func (d UploadDataInput) GetTopicId() string {
	return d.TopicId
}

func (d UploadDataInput) GetMemberId() string {
	return ""
}

func (d UploadDataInput) Origin() string {
	return origin.FindOrigin(d.SpaceId)
}
