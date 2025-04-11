package updates_topics

import "kasper/cmd/babble/sigma/api/model"

type Update struct {
	Topic model.Topic `json:"topic"`
}
