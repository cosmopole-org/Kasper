package outputs_points

import (
	"kasper/src/shell/api/model"
)

type CreateOutput struct {
	Point model.Point `json:"point"`
}
