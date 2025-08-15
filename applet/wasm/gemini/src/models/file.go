package model

type File struct {
	Id      string `json:"id" gorm:"primaryKey;column:id"`
	PointId string `json:"pointId" gorm:"column:topic_id"`
	OwnerId string `json:"senderId" gorm:"column:sender_id"`
}
