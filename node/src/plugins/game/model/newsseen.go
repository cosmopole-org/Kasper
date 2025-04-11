package game_model

type NewsSeen struct {
	Id      string `json:"id" gorm:"primaryKey;column:id"`
	Payload string `json:"payload" gorm:"column:payload"`
}

func (m NewsSeen) Type() string {
	return "NewsSeen"
}
