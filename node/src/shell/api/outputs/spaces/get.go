package outputs_spaces

import (
	models "kasper/src/shell/api/model"
)

type GetOutput struct {
	Space models.Space `json:"space"`
}
