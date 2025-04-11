package outputs_topics

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type DeleteOutput struct {
	Topic models.Topic `json:"topic"`
}
