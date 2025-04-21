package model

import (
	"kasper/src/abstract/models"
)

type File struct {
	Id      string `json:"id" gorm:"primaryKey;column:id"`
	TopicId string `json:"topicId" gorm:"column:topic_id"`
	OwnerId string `json:"senderId" gorm:"column:sender_id"`
}

func (d File) Type() string {
	return "File"
}

func (d File) Push(trx models.ITrx) {
	trx.PutObj(d.Type(), d.Id, map[string][]byte{
		"topicId": []byte(d.TopicId),
		"ownerId": []byte(d.OwnerId),
	})
}

func (d File) Pull(trx models.ITrx) File {
	m := trx.GetObj(d.Type(), d.Id)
	if len(m) > 0 {
		d.TopicId = string(m["topicId"])
		d.OwnerId = string(m["ownerId"])
	}
	return d
}
