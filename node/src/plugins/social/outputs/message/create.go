package outputs_message

import models "kasper/src/plugins/social/model"

type CreateMessageOutput struct {
	Message models.Message `json:"message"`
}
