package models

type MatchJoinPacket struct {
	GodMember Member `json:"godMember"`
	MyMember  Member `json:"myMember"`
	Space     Space  `json:"space"`
	Topic     Topic  `json:"topic"`
}