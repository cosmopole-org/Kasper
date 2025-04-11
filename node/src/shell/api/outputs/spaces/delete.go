package outputs_spaces

import (
	models "kasper/src/shell/api/model"
)

type DeleteOutput struct {
	Space models.Space `json:"space"`
}
