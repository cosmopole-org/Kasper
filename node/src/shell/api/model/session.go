package model

import "kasper/src/abstract/models"

type Session struct {
	Id     string `json:"id" gorm:"primaryKey;column:id"`
	UserId string `json:"userId" gorm:"column:user_id"`
}

func (d Session) Type() string {
	return "Session"
}

func (d Session) Push(trx models.ITrx) {
	trx.PutObj(d.Type(), d.Id, map[string][]byte{
		"userId": []byte(d.UserId),
	})
}

func (d Session) Pull(trx models.ITrx) Session {
	m := trx.GetObj(d.Type(), d.Id)
	if len(m) > 0 {
		d.UserId = string(m["userId"])
	}
	return d
}
