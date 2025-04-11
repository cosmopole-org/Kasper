package outputs_invites

import (
	models "kasper/src/shell/api/model"
)

type AcceptOutput struct {
	Member models.Member `json:"member"`
}
