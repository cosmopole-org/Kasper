package outputs_spaces

import (
	models "kasper/cmd/babble/sigma/api/model"
)

type AddMemberOutput struct {
	Member models.Member `json:"member"`
}
