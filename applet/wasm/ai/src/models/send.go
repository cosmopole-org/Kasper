package model

type Send struct {
	User   User  `json:"user"`
	Point  Point `json:"point"`
	Action string       `json:"action"`
	Data   string       `json:"data"`
}
