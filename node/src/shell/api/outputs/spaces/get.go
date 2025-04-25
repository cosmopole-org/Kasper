package outputs_points

import (
	models "kasper/src/shell/api/model"
)

type GetOutput struct {
	Point models.Point `json:"point"`
}
