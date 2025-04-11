package outputs_users

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type AuthenticateOutput struct {
	Authenticated bool              `json:"authenticated"`
	User          models.PublicUser `json:"user"`
}
