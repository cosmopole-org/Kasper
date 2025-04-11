package outputs_topics

import (
	models "kasper/src/shell/api/model"
)

type CreateOutput struct {
	Topic models.Topic `json:"topic"`
}
