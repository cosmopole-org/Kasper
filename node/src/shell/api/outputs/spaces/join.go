package outputs_spaces

import (
	models "kasper/src/shell/api/model"
)

type JoinOutput struct {
	Member models.Member `json:"member"`
}
