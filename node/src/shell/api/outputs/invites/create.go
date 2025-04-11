package outputs_invites

import (
	models "kasper/src/shell/api/model"
)

type CreateOutput struct {
	Invite models.Invite `json:"invite"`
}
