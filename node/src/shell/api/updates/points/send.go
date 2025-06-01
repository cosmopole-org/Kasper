package updates_points

import models "kasper/src/shell/api/model"

type Send struct {
	User   models.User  `json:"user"`
	Point  models.Point `json:"point"`
	Action string       `json:"action"`
	Data   string       `json:"data"`
}
