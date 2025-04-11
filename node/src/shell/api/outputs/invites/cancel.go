package outputs_invites

import (
	models "kasper/src/shell/api/model"
)

type CancelOutput struct {
	Invite models.Invite `json:"invite"`
}
