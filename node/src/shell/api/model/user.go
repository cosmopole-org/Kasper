package model

import (
	"kasper/src/abstract/models"
)

type User struct {
	Id        string `json:"id"`
	Typ       string `json:"type"`
	Username  string `json:"username"`
	PublicKey string `json:"publicKey"`
}

func (d User) Type() string {
	return "User"
}

func (d User) Push(trx models.ITrx) {
	trx.PutObj(d.Type(), d.Id, map[string][]byte{
		"type":      []byte(d.Typ),
		"username":  []byte(d.Username),
		"publicKey": []byte(d.PublicKey),
	})
}

func (d User) Pull(trx models.ITrx) User {
	m := trx.GetObj(d.Type(), d.Id)
	if len(m) > 0 {
		d.Typ = string(m["type"])
		d.Username = string(m["username"])
		d.PublicKey = string(m["publicKey"])
	}
	return d
}
