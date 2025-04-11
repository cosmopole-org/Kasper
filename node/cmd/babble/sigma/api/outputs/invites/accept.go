package outputs_invites

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type AcceptOutput struct {
	Member models.Member `json:"member"`
}
