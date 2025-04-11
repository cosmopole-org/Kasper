package outputs_users

import (
	models "kasper/src/shell/api/model"
)

type GetOutput struct {
	User models.PublicUser `json:"user"`
}
