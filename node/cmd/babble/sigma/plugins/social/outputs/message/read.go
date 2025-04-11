package outputs_message

import models "kasper/cmd/babble/sigma/plugins/social/model"

type ReadMessagesOutput struct {
	Messages []models.ResultMessage `json:"messages"`
}
