package model

import (
	"kasper/src/abstract/models/trx"
)

type File struct {
	Id      string `json:"id" gorm:"primaryKey;column:id"`
	PointId string `json:"pointId" gorm:"column:topic_id"`
	OwnerId string `json:"senderId" gorm:"column:sender_id"`
}

func (d File) Type() string {
	return "File"
}

func (d File) Push(trx trx.ITrx) {
	trx.PutObj(d.Type(), d.Id, map[string][]byte{
		"pointId": []byte(d.PointId),
		"ownerId": []byte(d.OwnerId),
	})
}

func (d File) Pull(trx trx.ITrx) File {
	m := trx.GetObj(d.Type(), d.Id)
	if len(m) > 0 {
		d.PointId = string(m["pointId"])
		d.OwnerId = string(m["ownerId"])
	}
	return d
}
