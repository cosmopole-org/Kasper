package outputs_users

import (
	"kasper/cmd/babble/sigma/api/model"
)

type LoginOutput struct {
	User       model.User    `json:"user"`
	Session    model.Session `json:"session"`
	PrivateKey string        `json:"privateKey"`
}
