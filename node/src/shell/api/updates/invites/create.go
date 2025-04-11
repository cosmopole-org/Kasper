package updates_invites

import "kasper/src/shell/api/model"

type Create struct {
	Invite model.Invite `json:"invite"`
}
