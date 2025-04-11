package outputs_users

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type DeleteOutput struct {
	User models.User `json:"user"`
}
