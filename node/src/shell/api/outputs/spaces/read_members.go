package outputs_points

import (
	models "kasper/src/shell/api/model"
)

type ReadMemberOutput struct {
	Members []models.User `json:"members"`
}
