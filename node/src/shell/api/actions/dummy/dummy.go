package actions_dummy

import (
	"kasper/src/abstract"
	"kasper/src/shell/api/inputs"
	"kasper/src/shell/layer1/adapters"
	"os"
	"time"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return nil
}

// Hello /api/hello check [ false false false ] access [ true false false false GET ]
func (a *Actions) Hello(_ abstract.IState, _ inputs.HelloInput) (any, error) {
	return `{ "hello": "world" }`, nil
}

// Time /api/time check [ false false false ] access [ true false false false GET ]
func (a *Actions) Time(_ abstract.IState, _ inputs.HelloInput) (any, error) {
	return map[string]any{"time": time.Now().UnixMilli()}, nil
}

// Ping /api/ping check [ false false false ] access [ true false false false GET ]
func (a *Actions) Ping(_ abstract.IState, _ inputs.HelloInput) (any, error) {
	return os.Getenv("MAIN_PORT"), nil
}
