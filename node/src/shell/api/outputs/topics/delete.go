package outputs_topics

import (
	models "kasper/src/shell/api/model"
)

type DeleteOutput struct {
	Topic models.Topic `json:"topic"`
}
