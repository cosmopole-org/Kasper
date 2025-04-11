package inputs_interact

type AcceptFriendRequestDto struct {
	UserId string `json:"userId" validate:"required"`
	Orig   string `json:"orig"`
}

func (d AcceptFriendRequestDto) GetSpaceId() string {
	return ""
}

func (d AcceptFriendRequestDto) GetTopicId() string {
	return ""
}

func (d AcceptFriendRequestDto) GetMemberId() string {
	return ""
}

func (d AcceptFriendRequestDto) Origin() string {
	return d.Orig
}