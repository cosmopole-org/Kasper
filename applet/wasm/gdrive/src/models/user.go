package model

type User struct {
	Id        string `json:"id"`
	Typ       string `json:"type"`
	Username  string `json:"username"`
	PublicKey string `json:"publicKey"`
}
