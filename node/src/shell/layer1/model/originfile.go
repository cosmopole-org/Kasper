package module_model

import (
	"kasper/src/shell/api/model"
	"mime/multipart"
)

type OriginFile struct {
	File      model.File
	UserId    string
	SpaceId   string
	TopicId   string
	RequestId string
	Data      *multipart.FileHeader
}
