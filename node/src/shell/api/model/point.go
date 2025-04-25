package model

import (
	"bytes"
	"kasper/src/abstract/models/trx"
	"log"
)

type Point struct {
	Id       string `json:"id"`
	PersHist bool   `json:"persHist"`
	IsPublic bool   `json:"isPublic"`
}

func (d Point) Type() string {
	return "Point"
}

func (d Point) Push(trx trx.ITrx) {
	b := byte(0x00)
	if d.IsPublic {
		b = byte(0x01)
	}
	trx.PutObj(d.Type(), d.Id, map[string][]byte{
		"isPublic": []byte{b},
	})
}

func (d Point) Pull(trx trx.ITrx) Point {
	m := trx.GetObj(d.Type(), d.Id)
	if len(m) > 0 {
		d.IsPublic = bytes.Equal(m["isPublic"], []byte{0x01})
	}
	return d
}

func (d Point) List(trx trx.ITrx, prefix string, positional ...int) ([]Point, error) {
	offset := -1
	count := -1
	if len(positional) == 1 {
		offset = positional[0]
	}
	if len(positional) == 2 {
		count = positional[1]
	}
	list, err := trx.GetLinksList(prefix, offset, count)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	for i := 0; i < len(list); i++ {
		list[i] = list[i][len(prefix):]
	}
	objs, err := trx.GetObjList("Point", list)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	entities := []Point{}
	for id, m := range objs {
		if len(m) > 0 {
			d := Point{}
			d.Id = id
			d.IsPublic = bytes.Equal(m["isPublic"], []byte{0x01})
			entities = append(entities, d)
		}
	}
	return entities, nil
}
