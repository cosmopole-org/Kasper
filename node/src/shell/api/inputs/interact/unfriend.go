package inputs_interact

type UnfriendUserDto struct {
	UserId string `json:"userId" validate:"required"`
	Orig   string `json:"orig"`
}

func (d UnfriendUserDto) GetSpaceId() string {
	return ""
}

func (d UnfriendUserDto) GetTopicId() string {
	return ""
}

func (d UnfriendUserDto) GetMemberId() string {
	return ""
}

func (d UnfriendUserDto) Origin() string {
	return d.Orig
}
