package inputs_message

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

type CreateMessageInput struct {
	TopicId  string `json:"topicId" validate:"required"`
	MemberId string `json:"memberId"`
	Data     Json   `json:"data" validate:"required"`
}

func (d CreateMessageInput) GetSpaceId() string {
	return ""
}

func (d CreateMessageInput) GetTopicId() string {
	return d.TopicId
}

func (d CreateMessageInput) GetMemberId() string {
	return d.MemberId
}

func (d CreateMessageInput) Origin() string {
	return ""
}
