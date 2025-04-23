package actions_dummy

import (
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/state"
	"kasper/src/shell/api/inputs"
	"os"
	"time"
)

type Actions struct {
	app core.ICore
}

func Install(a *Actions) error {
	return nil
}

// Hello /api/hello check [ false false false ] access [ true false false false GET ]
func (a *Actions) Hello(_ state.IState, _ inputs.HelloInput) (any, error) {
	return `{ "hello": "world" }`, nil
}

// Time /api/time check [ false false false ] access [ true false false false GET ]
func (a *Actions) Time(_ state.IState, _ inputs.HelloInput) (any, error) {
	return map[string]any{"time": time.Now().UnixMilli()}, nil
}

// Ping /api/ping check [ false false false ] access [ true false false false GET ]
func (a *Actions) Ping(_ state.IState, _ inputs.HelloInput) (any, error) {
	return os.Getenv("MAIN_PORT"), nil
}
