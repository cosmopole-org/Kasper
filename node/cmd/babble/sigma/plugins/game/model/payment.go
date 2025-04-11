package game_model

type Payment struct {
	Id      string `json:"id" gorm:"primaryKey;column:id"`
	UserId  string `json:"userId" gorm:"column:user_id"`
	Product string `json:"product" gorm:"column:product"`
	GameKey string `json:"gameKey" gorm:"column:game_key"`
	Time    int64  `json:"time" gorm:"column:time"`
	Market  string `json:"market" gorm:"column:market"`
}

func (m Payment) Type() string {
	return "Payment"
}
