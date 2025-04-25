package outputs_points

import (
	models "kasper/src/shell/api/model"
)

type DeleteOutput struct {
	Point models.Point `json:"point"`
}
