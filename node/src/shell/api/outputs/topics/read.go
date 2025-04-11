package outputs_topics

import (
	models "kasper/src/shell/api/model"
)

type ReadOutput struct {
	Topics []models.Topic `json:"topics"`
}
