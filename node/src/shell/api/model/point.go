package model

import (
	"bytes"
	"kasper/src/abstract/models"
)

type Point struct {
	Id       string `json:"id"`
	IsPublic bool   `json:"isPublic"`
}

func (d Point) Type() string {
	return "Point"
}

func (d Point) Push(trx models.ITrx) {
	b := byte(0x00)
	if d.IsPublic {
		b = byte(0x01)
	}
	trx.PutObj(d.Type(), d.Id, map[string][]byte{
		"isPublic": []byte{b},
	})
}

func (d Point) Pull(trx models.ITrx) Point {
	m := trx.GetObj(d.Type(), d.Id)
	if len(m) > 0 {
		d.IsPublic = bytes.Equal(m["isPublic"], []byte{0x01})
	}
	return d
}
