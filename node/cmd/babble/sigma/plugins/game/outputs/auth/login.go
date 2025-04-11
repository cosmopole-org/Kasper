package game_outputs_auth

type LoginOutput struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}
