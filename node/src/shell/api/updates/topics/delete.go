package updates_topics

import "kasper/src/shell/api/model"

type Delete struct {
	Topic model.Topic `json:"topic"`
}
