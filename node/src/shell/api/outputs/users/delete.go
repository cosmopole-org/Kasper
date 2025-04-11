package outputs_users

import (
	models "kasper/src/shell/api/model"
)

type DeleteOutput struct {
	User models.User `json:"user"`
}
