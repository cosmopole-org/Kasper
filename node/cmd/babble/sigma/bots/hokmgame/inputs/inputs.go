package inputs

import "kasper/cmd/babble/sigma/bots/hokmgame/models"

type HelloInput struct {
	Name string `json:"name"`
}

type HelloOutput struct {
	Message string `json:"message"`
}

type ByeInput struct{}

type ByeOutput struct {
	Message string `json:"message"`
}

// game

type CreateGameInput struct {
	Turns      float64         `json:"turns"`
	Level      string          `json:"level"`
	Players    []models.Player `json:"players"`
	IsFriendly bool            `json:"isFriendly"`
}

type SpecifyHokmInput struct {
	Hokm string `json:"hokm"`
}

type PlayGameInput struct {
	Action string `json:"action"`
}
