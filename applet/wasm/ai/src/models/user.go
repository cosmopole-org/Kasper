package model

type User struct {
	Id        string         `json:"id" gorm:"primaryKey;column:id"`
	Number    int            `json:"number" gorm:"uniqueIndex;autoIncrement;column:number"`
	Typ       string         `json:"typ" gorm:"column:type"`
	Username  string         `json:"username" gorm:"column:username"`
	Name      string         `json:"name" gorm:"column:name"`
	Avatar    string         `json:"avatar" gorm:"column:avatar"`
	PublicKey string         `json:"publicKey" gorm:"column:public_key"`
}

func (d User) Type() string {
	return "User"
}
