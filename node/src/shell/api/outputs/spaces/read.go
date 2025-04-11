package outputs_spaces

import (
	models "kasper/src/shell/api/model"
)

type ReadOutput struct {
	Spaces []models.Space `json:"spaces"`
}
