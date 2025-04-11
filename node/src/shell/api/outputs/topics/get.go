package outputs_topics

import (
	models "kasper/src/shell/api/model"
)

type GetOutput struct {
	Topic models.Topic `json:"topic"`
}
