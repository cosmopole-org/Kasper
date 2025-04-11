package outputs_spaces

import "kasper/src/shell/api/model"

type CreateSpaceOutput struct {
	Space  model.Space  `json:"space"`
	Topic  model.Topic  `json:"topic"`
	Member model.Member `json:"member"`
}
