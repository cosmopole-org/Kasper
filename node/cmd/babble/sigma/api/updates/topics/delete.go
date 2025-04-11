package updates_topics

import "kasper/cmd/babble/sigma/api/model"

type Delete struct {
	Topic model.Topic `json:"topic"`
}
