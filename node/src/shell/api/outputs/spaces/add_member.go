package outputs_spaces

import (
	models "kasper/src/shell/api/model"
)

type AddMemberOutput struct {
	Member models.Member `json:"member"`
}
