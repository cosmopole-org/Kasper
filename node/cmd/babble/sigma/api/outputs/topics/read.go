package outputs_topics

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type ReadOutput struct {
	Topics []models.Topic `json:"topics"`
}
