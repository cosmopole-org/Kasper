package inputs_interact

type SendFriendRequestDto struct {
	Code   string `json:"code"`
	UserId string `json:"userId"`
	Orig   string `json:"orig"`
}

func (d SendFriendRequestDto) GetSpaceId() string {
	return ""
}

func (d SendFriendRequestDto) GetTopicId() string {
	return ""
}

func (d SendFriendRequestDto) GetMemberId() string {
	return ""
}

func (d SendFriendRequestDto) Origin() string {
	return d.Orig
}