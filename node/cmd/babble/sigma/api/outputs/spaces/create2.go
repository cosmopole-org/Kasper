package outputs_spaces

import "kasper/cmd/babble/sigma/api/model"

type CreateSpaceOutput struct {
	Space  model.Space  `json:"space"`
	Topic  model.Topic  `json:"topic"`
	Member model.Member `json:"member"`
}
