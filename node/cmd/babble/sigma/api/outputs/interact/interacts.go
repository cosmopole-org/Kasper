package outputs_interact

import models "kasper/cmd/babble/sigma/api/model"

type InteractsOutput struct {
	Interactions []*models.PreparedInteraction `json:"interactions"`
}
