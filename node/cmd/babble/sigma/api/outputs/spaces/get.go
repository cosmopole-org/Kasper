package outputs_spaces

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type GetOutput struct {
	Space models.Space `json:"space"`
}
