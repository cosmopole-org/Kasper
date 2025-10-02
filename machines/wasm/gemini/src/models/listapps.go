package model

type ListPointAppsInput struct {
	PointId string `json:"pointId" validate:"required"`
}
