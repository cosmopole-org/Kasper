package outputs_spaces

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type JoinOutput struct {
	Member models.Member `json:"member"`
}
