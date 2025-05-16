package outputs_points

import (
	models "kasper/src/shell/api/model"
)

type ReadOutput struct {
	Points []models.Point `json:"points"`
}
