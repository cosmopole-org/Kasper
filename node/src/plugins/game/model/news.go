package game_model

type News struct {
	Id      string `json:"id" gorm:"primaryKey;column:id"`
	Data    Json   `json:"data" gorm:"column:data"`
	GameKey string `json:"gameKey" gorm:"column:game_key"`
	Time    int64  `json:"time" gorm:"column:time"`
}

func (m News) Type() string {
	return "News"
}
