package updates_spaces

import "kasper/cmd/babble/sigma/api/model"

type Update struct {
	Space model.Space `json:"space"`
}
