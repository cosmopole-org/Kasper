package model

import (
	"encoding/binary"
	"kasper/src/abstract/models/trx"
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

type Vm struct {
	MachineId string `json:"id"`
	AppId     string `json:"appId"`
	Runtime   string `json:"runtime"`
}

func (m Vm) Type() string {
	return "Vm"
}

func (d Vm) Push(trx trx.ITrx) {
	trx.PutObj(d.Type(), d.MachineId, map[string][]byte{
		"machineId": []byte(d.MachineId),
		"appId":     []byte(d.AppId),
		"runtime":   []byte(d.Runtime),
	})
}

func (d Vm) Pull(trx trx.ITrx) Vm {
	m := trx.GetObj(d.Type(), d.MachineId)
	if len(m) > 0 {
		d.MachineId = string(m["machineId"])
		d.AppId = string(m["appId"])
		d.Runtime = string(m["runtime"])
	}
	return d
}
