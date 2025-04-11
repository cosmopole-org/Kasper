package updates_topics

import models "kasper/src/shell/api/model"

type Send struct {
	User         models.User   `json:"user"`
	Topic        models.Topic  `json:"topic"`
	Member       models.Member `json:"member"`
	TargetMember models.Member `json:"targetMember"`
	Action       string        `json:"action"`
	Data         string        `json:"data"`
}
