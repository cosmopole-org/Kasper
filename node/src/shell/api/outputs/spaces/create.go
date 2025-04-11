package outputs_spaces

import (
	"kasper/src/shell/api/model"
)

type CreateOutput struct {
	Space  model.Space  `json:"space"`
	Member model.Member `json:"member"`
	Topic  model.Topic  `json:"topic"`
}
