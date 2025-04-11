package updates_spaces

import "kasper/src/shell/api/model"

type AddMember struct {
	SpaceId string       `json:"spaceId"`
	TopicId string       `json:"topicId"`
	Member  model.Member `json:"member"`
}
