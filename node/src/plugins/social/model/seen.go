package model

type Seen struct {
	Id        string `json:"id" gorm:"primaryKey;column:id"`
	ChatId    string `json:"chatId" gorm:"column:chat_id"`
	LastMsgId string `json:"lastMsgId" gorm:"column:last_msg_id"`
	UserId    string `json:"userId" gorm:"column:user_id"`
}

func (m Seen) Type() string {
	return "Seen"
}
