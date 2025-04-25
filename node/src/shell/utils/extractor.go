package utils

import (
	"kasper/src/abstract/models/action"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/input"
	"kasper/src/abstract/state"
	mainaction "kasper/src/core/module/actor/model/base"
	"kasper/src/core/module/actor/model/secured"
	"kasper/src/shell/utils/vaidate"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func ExtractAction[T input.IInput](app core.ICore, actionFunc func(state.IState, T) (any, error)) action.IAction {
	key, _ := ExtractActionMetadata(actionFunc)
	action := mainaction.NewAction(app.ModifyState, key, func(state state.IState, input input.IInput) (any, error) {
		return actionFunc(state, input.(T))
	})
	return action
}

func ExtractSecureAction[T input.IInput](app core.ICore, actionFunc func(state.IState, T) (any, error)) action.IAction {
	key, guard := ExtractActionMetadata(actionFunc)
	action := mainaction.NewAction(app.ModifyState, key, func(state state.IState, input input.IInput) (any, error) {
		return actionFunc(state, input.(T))
	})
	return secured.NewSecureAction(action, guard, app, map[string]actor.Parse{
		"http": func(i interface{}) (abstract.IInput, error) {
			input, err := net_http.ParseInput[T](i.(*fiber.Ctx))
			if err == nil {
				err2 := vaidate.Validate.Struct(input)
				if err2 == nil {
					return input, nil
				}
				return nil, err2
			}
			return nil, err
		},
		"push": func(i interface{}) (abstract.IInput, error) {
			input, err := net_pusher.ParseInput[T](i.(string))
			if err == nil {
				err2 := vaidate.Validate.Struct(input)
				if err2 == nil {
					return input, nil
				}
				return nil, err2
			}
			return nil, err
		},
		"grpc": func(i interface{}) (abstract.IInput, error) {
			input, err := net_grpc.ParseInput[T](i)
			if err == nil {
				err2 := vaidate.Validate.Struct(input)
				if err2 == nil {
					return input, nil
				}
				return nil, err2
			}
			return nil, err
		},
		"fed": func(i interface{}) (abstract.IInput, error) {
			input, err := net_federation.ParseInput[T](i.(string))
			if err == nil {
				err2 := vaidate.Validate.Struct(input)
				if err2 == nil {
					return input, nil
				}
				return nil, err2
			}
			return nil, err
		},
	})
}

func ExtractActionMetadata(function interface{}) (string, *actor.Guard) {
	var ts = strings.Split(FuncDescription(function), " ")
	var tokens []string
	for _, token := range ts {
		if len(strings.Trim(token, " ")) > 0 {
			tokens = append(tokens, token)
		}
	}
	var key = tokens[0]
	var guard *actor.Guard
	if tokens[1] == "check" && tokens[2] == "[" && tokens[6] == "]" {
		guard = &actor.Guard{IsUser: tokens[3] == "true", IsInSpace: tokens[4] == "true", IsInTopic: tokens[5] == "true"}
		//if tokens[7] == "access" && tokens[8] == "[" && tokens[14] == "]" {
		//	access = Access{Http: tokens[9] == "true", Ws: tokens[10] == "true", Grpc: tokens[11] == "true", Fed: tokens[12] == "true", ActionType: tokens[13]}
		//}
	}
	return key, guard
}
