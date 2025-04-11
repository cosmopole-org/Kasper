package outputs_machiner

import (
	"kasper/cmd/babble/sigma/api/model"
)

type CreateOutput struct {
	User    model.User    `json:"user"`
}
