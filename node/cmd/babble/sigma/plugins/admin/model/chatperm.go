package admin_model

type ChatPerm struct {
	Id     string `json:"id" gorm:"primaryKey;column:id"`
	UserId string `json:"userId" gorm:"column:user_id"`
	Time   int64  `json:"time" gorm:"column:time"`
}

func (m ChatPerm) Type() string {
	return "ChatPerm"
}
