package outputs_spaces

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type DeleteOutput struct {
	Space models.Space `json:"space"`
}
