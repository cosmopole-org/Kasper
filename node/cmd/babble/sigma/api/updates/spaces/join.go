package updates_spaces

import "kasper/cmd/babble/sigma/api/model"

type Join struct {
	Member model.Member `json:"member"`
}
