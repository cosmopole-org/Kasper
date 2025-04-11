package inputs_interact

type DeclineFriendRequestDto struct {
	UserId string `json:"userId" validate:"required"`
	Orig   string `json:"orig"`
}

func (d DeclineFriendRequestDto) GetSpaceId() string {
	return ""
}

func (d DeclineFriendRequestDto) GetTopicId() string {
	return ""
}

func (d DeclineFriendRequestDto) GetMemberId() string {
	return ""
}

func (d DeclineFriendRequestDto) Origin() string {
	return d.Orig
}