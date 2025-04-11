package outputs_users

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type GetOutput struct {
	User models.PublicUser `json:"user"`
}
