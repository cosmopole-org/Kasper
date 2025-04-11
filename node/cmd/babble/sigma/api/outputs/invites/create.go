package outputs_invites

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type CreateOutput struct {
	Invite models.Invite `json:"invite"`
}
