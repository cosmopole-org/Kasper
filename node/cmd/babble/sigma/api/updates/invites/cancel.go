package updates_invites

import "kasper/cmd/babble/sigma/api/model"

type Cancel struct {
	Invite model.Invite `json:"invite"`
}