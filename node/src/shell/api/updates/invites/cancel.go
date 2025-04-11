package updates_invites

import "kasper/src/shell/api/model"

type Cancel struct {
	Invite model.Invite `json:"invite"`
}
