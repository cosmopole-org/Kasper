package model

type Key struct {
	Id         string `json:"id" gorm:"primaryKey;column:id"`
	PublicKey  string `json:"pubkey" gorm:"column:pubkey"`
	PrivateKey string `json:"privkey" gorm:"column:privkey"`
}

func (d Key) Type() string {
	return "Key"
}
