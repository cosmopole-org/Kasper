package updates_spaces

import "kasper/cmd/babble/sigma/api/model"

type AddMember struct {
	SpaceId string `json:"spaceId"`
	TopicId string `json:"topicId"`
	Member  model.Member    `json:"member"`
}
