package outputs_message

import models "kasper/src/plugins/social/model"

type ReadMessagesOutput struct {
	Messages []models.ResultMessage `json:"messages"`
}
