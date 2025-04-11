package updates_spaces

import "kasper/src/shell/api/model"

type Delete struct {
	Space model.Space `json:"space"`
}
