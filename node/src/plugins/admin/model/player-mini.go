package admin_model

type PlayerMini struct {
	Id         string `json:"id"`
	Coin       any    `json:"coin"`
	Gem        any    `json:"gem"`
	Energy     any    `json:"energy"`
	Email      string `json:"email"`
	PlayerName string `json:"playerName"`
}
