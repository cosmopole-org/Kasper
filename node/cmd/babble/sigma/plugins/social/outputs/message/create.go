package outputs_message

import models "kasper/cmd/babble/sigma/plugins/social/model"

type CreateMessageOutput struct {
	Message models.Message `json:"message"`
}
