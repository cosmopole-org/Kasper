package outputs_invites

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type CancelOutput struct {
	Invite models.Invite `json:"invite"`
}
