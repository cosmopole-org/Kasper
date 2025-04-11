package updates_invites

import "kasper/cmd/babble/sigma/api/model"

type Create struct {
	Invite model.Invite `json:"invite"`
}
