package model

import (
	"database/sql/driver"
	"encoding/json"
)

type Json map[string]interface{}

func (j Json) Value() (driver.Value, error) {
	b, err := json.Marshal(j)
	return string(b), err
}
func (j *Json) Scan(value interface{}) error {
	if v, ok := value.(string); ok {
		return json.Unmarshal([]byte(v), &j)
	} else if v, ok := value.([]byte); ok {
		return json.Unmarshal(v, &j)
	} else {
		return json.Unmarshal([]byte(value.(string)), &j)
	}
}

type Report struct {
	Id         string `json:"id" gorm:"primaryKey;column:id"`
	ReporterId string `json:"reporterId" gorm:"column:reporter_id"`
	GameKey    string `json:"gameKey" gorm:"column:gamekey"`
	Data       Json   `json:"data" gorm:"column:data"`
}

func (m Report) Type() string {
	return "Message"
}
