package outputs_spaces

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type ReadOutput struct {
	Spaces []models.Space `json:"spaces"`
}
