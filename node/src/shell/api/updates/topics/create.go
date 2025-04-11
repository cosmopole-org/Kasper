package updates_topics

import "kasper/src/shell/api/model"

type Create struct {
	Topic model.Topic `json:"topic"`
}
