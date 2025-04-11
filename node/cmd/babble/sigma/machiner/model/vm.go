package model

type Vm struct {
	MachineId string `json:"id" gorm:"primaryKey;column:id"`
	OwnerId   string `json:"ownerId" gorm:"column:owner_id"`
	Runtime   string `json:"runtime" gorm:"column:runtime"`
}

func (m Vm) Type() string {
	return "Vm"
}
