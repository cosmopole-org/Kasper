package model

import (
	"kasper/src/abstract/models/trx"
	"log"
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

func (d User) Push(trx trx.ITrx) {
	trx.PutObj(d.Type(), d.Id, map[string][]byte{
		"type":      []byte(d.Typ),
		"username":  []byte(d.Username),
		"publicKey": []byte(d.PublicKey),
	})
	trx.PutIndex("User", "username", "id", d.Username, []byte(d.Id))
}

func (d User) Pull(trx trx.ITrx) User {
	m := trx.GetObj(d.Type(), d.Id)
	if len(m) > 0 {
		d.Typ = string(m["type"])
		d.Username = string(m["username"])
		d.PublicKey = string(m["publicKey"])
	}
	return d
}

func (d User) List(trx trx.ITrx, prefix string) ([]User, error) {
	list, err := trx.GetLinksList(prefix, -1, -1)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	for i := 0; i < len(list); i++ {
		list[i] = list[i][len(prefix):]
	}
	objs, err := trx.GetObjList("User", list)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	entities := []User{}
	for id, m := range objs {
		if len(m) > 0 {
			d := User{}
			d.Id = id
			d.Typ = string(m["type"])
			d.Username = string(m["username"])
			d.PublicKey = string(m["publicKey"])
			entities = append(entities, d)
		}
	}
	return entities, nil
}
