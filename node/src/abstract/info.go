package abstract

type IInfo interface {
	IsGod() bool
	UserId() string
	MemberId() string
	SpaceId() string
	TopicId() string
	Identity() (string, string)
}
