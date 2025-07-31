package model

import (
	"encoding/binary"
	"kasper/src/abstract/models/trx"
	"log"
	"sort"
)

type App struct {
	Id      string `json:"id"`
	ChainId int64  `json:"chainId"`
	OwnerId string `json:"ownerId"`
}

func (m App) Type() string {
	return "App"
}

func (d App) Push(trx trx.ITrx) {
	cidBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(cidBytes, uint64(d.ChainId))
	trx.PutObj(d.Type(), d.Id, map[string][]byte{
		"id":      []byte(d.Id),
		"ownerId": []byte(d.OwnerId),
		"chainId": cidBytes,
	})
}

func (d App) Pull(trx trx.ITrx) App {
	m := trx.GetObj(d.Type(), d.Id)
	if len(m) > 0 {
		d.Id = string(m["id"])
		d.OwnerId = string(m["ownerId"])
		d.ChainId = int64(binary.LittleEndian.Uint64(m["chainId"]))
	}
	return d
}

func (d App) All(trx trx.ITrx, offset int64, count int64) ([]App, error) {
	objs, err := trx.GetObjList("App", []string{"*"}, map[string]string{}, offset, count)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	entities := []App{}
	for id, m := range objs {
		if len(m) > 0 {
			d := App{}
			d.Id = id
			d.OwnerId = string(m["ownerId"])
			d.ChainId = int64(binary.LittleEndian.Uint64(m["chainId"]))
			entities = append(entities, d)
		}
	}
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].Id < entities[j].Id
	})
	return entities, nil
}

type Vm struct {
	MachineId string `json:"id"`
	AppId     string `json:"appId"`
	Runtime   string `json:"runtime"`
	Path      string `json:"path"`
}

func (m Vm) Type() string {
	return "Vm"
}

func (d Vm) Push(trx trx.ITrx) {
	trx.PutObj(d.Type(), d.MachineId, map[string][]byte{
		"machineId": []byte(d.MachineId),
		"appId":     []byte(d.AppId),
		"runtime":   []byte(d.Runtime),
		"path":      []byte(d.Path),
	})
}

func (d Vm) Pull(trx trx.ITrx) Vm {
	m := trx.GetObj(d.Type(), d.MachineId)
	if len(m) > 0 {
		d.MachineId = string(m["machineId"])
		d.AppId = string(m["appId"])
		d.Runtime = string(m["runtime"])
		d.Path = string(m["path"])
	}
	return d
}

func (d Vm) All(trx trx.ITrx, offset int64, count int64) ([]Vm, error) {
	objs, err := trx.GetObjList("Vm", []string{"*"}, map[string]string{}, offset, count)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	entities := []Vm{}
	for id, m := range objs {
		if len(m) > 0 {
			d := Vm{}
			d.MachineId = id
			d.AppId = string(m["appId"])
			d.Runtime = string(m["runtime"])
			d.Path = string(m["path"])
			entities = append(entities, d)
		}
	}
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].MachineId < entities[j].MachineId
	})
	return entities, nil
}
