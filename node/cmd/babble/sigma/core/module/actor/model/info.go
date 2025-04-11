package module_actor_model

import "strings"

type Info struct {
	isGod   bool
	userId  string
	spaceId string
	topicId string
	memberId string
}

func NewInfo(userId string, spaceId string, topicId string, memberId string) *Info {
	return &Info{isGod: false, userId: userId, spaceId: spaceId, topicId: topicId, memberId: memberId}
}

func NewGodInfo(userId string, spaceId string, topicId string, isGod bool, memberId string) *Info {
	return &Info{isGod: isGod, userId: userId, spaceId: spaceId, topicId: topicId, memberId: memberId}
}

func (info *Info) IsGod() bool {
	return info.isGod
}

func (info *Info) UserId() string {
	return info.userId
}

func (info *Info) SpaceId() string {
	return info.spaceId
}

func (info *Info) TopicId() string {
	return info.topicId
}

func (info *Info) MemberId() string {
	return info.memberId
}

func (info *Info) Identity() (string, string) {
	identity := strings.Split(info.userId, "@")
	if len(identity) == 2 {
		return identity[0], identity[1]
	}
	return "", ""
}
