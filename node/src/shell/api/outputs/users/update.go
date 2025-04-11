package outputs_users

import (
	models "kasper/src/shell/api/model"
)

type UpdateOutput struct {
	User models.PublicUser `json:"user"`
}
