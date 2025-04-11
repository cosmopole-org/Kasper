package outputs_spaces

import (
	models "kasper/src/shell/api/model"
)

type UpdateOutput struct {
	Space models.Space `json:"space"`
}
