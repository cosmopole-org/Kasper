package outputs_topics

import (
	models "kasper/src/shell/api/model"
)

type UpdateOutput struct {
	Topic models.Topic `json:"topic"`
}
