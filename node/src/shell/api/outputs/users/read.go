package outputs_users

import (
	models "kasper/src/shell/api/model"
)

type ReadOutput struct {
	Users []models.PublicUser `json:"users"`
}
