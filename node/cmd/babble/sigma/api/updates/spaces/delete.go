package updates_spaces

import "kasper/cmd/babble/sigma/api/model"

type Delete struct {
	Space model.Space `json:"space"`
}
