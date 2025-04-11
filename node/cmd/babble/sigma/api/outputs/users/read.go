package outputs_users

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type ReadOutput struct {
	Users []models.PublicUser `json:"users"`
}
