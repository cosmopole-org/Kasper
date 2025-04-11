package outputs_spaces

import (
	"kasper/cmd/babble/sigma/api/model"
)

type CreateOutput struct {
	Space  model.Space  `json:"space"`
	Member model.Member `json:"member"`
	Topic  model.Topic  `json:"topic"`
}
