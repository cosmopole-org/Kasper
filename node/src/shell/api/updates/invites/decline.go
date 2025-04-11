package updates_invites

import "kasper/src/shell/api/model"

type Decline struct {
	Invite model.Invite `json:"invite"`
}
