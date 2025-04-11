package outputs_users

import (
	models "kasper/src/shell/api/model"
)

type AuthenticateOutput struct {
	Authenticated bool              `json:"authenticated"`
	User          models.PublicUser `json:"user"`
}
