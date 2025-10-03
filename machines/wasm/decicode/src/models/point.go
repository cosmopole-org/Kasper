package model

type Point struct {
	Id       string `json:"id"`
	PersHist bool   `json:"persHist"`
	IsPublic bool   `json:"isPublic"`
}
