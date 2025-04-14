package model

type Topic struct {
	Id        string `json:"id" gorm:"primaryKey;column:id"`
	Title     string `json:"title" gorm:"column:title"`
	Avatar    string `json:"avatar" gorm:"column:avatar"`
	SpaceId   string `json:"spaceId" gorm:"column:space_id"`
	IsPrivate bool   `json:"isPrivate" gorm:"column:is_private"`
}

func (d Topic) Type() string {
	return "Topic"
}
