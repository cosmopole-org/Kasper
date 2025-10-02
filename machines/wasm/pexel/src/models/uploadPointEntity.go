package model

type UploadPointEntityInput struct {
	Data     string `json:"data" validate:"required"`
	PointId string `json:"pointId" validate:"required"`
	EntityId string `json:"entityId" validate:"required"`
}
