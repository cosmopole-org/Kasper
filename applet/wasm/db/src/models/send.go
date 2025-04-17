package model

type Send struct {
	User         User   `json:"user"`
	Topic        Topic  `json:"topic"`
	Member       Member `json:"member"`
	TargetMember Member `json:"targetMember"`
	Action       string        `json:"action"`
	Data         string        `json:"data"`
}
