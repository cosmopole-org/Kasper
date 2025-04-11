package outputs_interact

import models "kasper/src/shell/api/model"

type InteractsOutput struct {
	Interactions []*models.PreparedInteraction `json:"interactions"`
}
