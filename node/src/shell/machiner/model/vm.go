package model

import (
	"kasper/src/abstract/models/trx"
)

type Vm struct {
	MachineId string `json:"id" gorm:"primaryKey;column:id"`
	OwnerId   string `json:"ownerId" gorm:"column:owner_id"`
	Runtime   string `json:"runtime" gorm:"column:runtime"`
}

func (m Vm) Type() string {
	return "Vm"
}

func (d Vm) Push(trx trx.ITrx) {
	trx.PutObj(d.Type(), d.MachineId, map[string][]byte{
		"machineId": []byte(d.MachineId),
		"ownerId":   []byte(d.OwnerId),
		"runtime":   []byte(d.Runtime),
	})
}

func (d Vm) Pull(trx trx.ITrx) Vm {
	m := trx.GetObj(d.Type(), d.MachineId)
	if len(m) > 0 {
		d.MachineId = string(m["machineId"])
		d.OwnerId = string(m["ownerId"])
		d.Runtime = string(m["runtime"])
	}
	return d
}
