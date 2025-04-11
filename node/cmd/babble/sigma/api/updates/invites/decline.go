package updates_invites

import "kasper/cmd/babble/sigma/api/model"

type Decline struct {
	Invite model.Invite `json:"invite"`
}
