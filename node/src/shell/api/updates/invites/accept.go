package updates_invites

import "kasper/src/shell/api/model"

type Accept struct {
	Invite model.Invite `json:"invite"`
}
