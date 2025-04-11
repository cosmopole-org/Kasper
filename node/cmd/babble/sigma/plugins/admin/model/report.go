package admin_model

type Report struct {
	Id         string `json:"id" gorm:"primaryKey;column:id"`
	ReporterId string `json:"reporterId" gorm:"column:reporter_id"`
	GameKey    string `json:"gameKey" gorm:"column:gamekey"`
	Data       Json   `json:"data" gorm:"column:data"`
}

func (m Report) Type() string {
	return "Message"
}

type ResultReport struct {
	Id         string `json:"id"`
	ReporterId string `json:"reporterId"`
	GameKey    string `json:"gameKey"`
	Data       any    `json:"data"`
}
