package updates_topics

import "kasper/cmd/babble/sigma/api/model"

type Create struct {
	Topic model.Topic `json:"topic"`
}
