package actions_invite

import (
	"errors"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/state"
	inputsinvites "kasper/src/shell/api/inputs/invites"
	"kasper/src/shell/api/model"
	outputsinvites "kasper/src/shell/api/outputs/invites"
	updatesinvites "kasper/src/shell/api/updates/invites"
	updates_points "kasper/src/shell/api/updates/points"
	"kasper/src/shell/utils/future"
	"log"
)

type Actions struct {
	app core.ICore
}

func Install(a *Actions) error {
	return nil
}

// Create /invites/create check [ true true false ] access [ true false false false POST ]
func (a *Actions) Create(state state.IState, input inputsinvites.CreateInput) (any, error) {
	trx := state.Trx()
	if trx.GetLink("admin::"+state.Info().PointId()+"::"+state.Info().UserId()) != "true" {
		return nil, errors.New("you are not admin")
	}
	if !trx.HasObj("Point", state.Info().PointId()) {
		return nil, errors.New("point not found")
	}
	point := model.Point{Id: state.Info().PointId()}.Pull(trx)
	trx.PutLink("invite::"+point.Id+"::"+input.UserId, "true")
	future.Async(func() {
		a.app.Tools().Signaler().SignalUser("invites/create", "", input.UserId, updatesinvites.Create{PointId: point.Id}, true)
	}, false)
	return outputsinvites.CreateOutput{}, nil
}

// Cancel /invites/cancel check [ true true false ] access [ true false false false POST ]
func (a *Actions) Cancel(state state.IState, input inputsinvites.CancelInput) (any, error) {
	trx := state.Trx()
	if trx.GetLink("admin::"+state.Info().PointId()+"::"+state.Info().UserId()) != "true" {
		return nil, errors.New("you are not admin")
	}
	if trx.GetLink("invite::"+state.Info().PointId()+"::"+input.UserId) != "true" {
		return nil, errors.New("invitation does not exist")
	}
	trx.DelKey("link::invite::" + state.Info().PointId() + "::" + input.UserId)
	future.Async(func() {
		a.app.Tools().Signaler().SignalUser("invites/cancel", "", input.UserId, updatesinvites.Cancel{PointId: state.Info().PointId()}, true)
	}, false)
	return outputsinvites.CancelOutput{}, nil
}

// Accept /invites/accept check [ true false false ] access [ true false false false POST ]
func (a *Actions) Accept(state state.IState, input inputsinvites.AcceptInput) (any, error) {
	trx := state.Trx()
	if trx.GetLink("invite::"+input.PointId+"::"+state.Info().UserId()) != "true" {
		return nil, errors.New("invitation does not exist")
	}
	trx.DelKey("link::invite::" + input.PointId + "::" + state.Info().UserId())
	trx.PutLink("member::"+input.PointId+"::"+state.Info().UserId(), "true")
	a.app.Tools().Signaler().JoinGroup(input.PointId, state.Info().UserId())
	admins, err := trx.GetLinksList("admin::"+input.PointId+"::", -1, -1)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	for _, admin := range admins {
		future.Async(func() {
			a.app.Tools().Signaler().SignalUser("invites/accept", "", admin, updatesinvites.Accept{UserId: state.Info().UserId(), PointId: input.PointId}, true)
		}, false)
	}
	user := model.User{Id: state.Info().UserId()}.Pull(trx)
	future.Async(func() {
		a.app.Tools().Signaler().SignalGroup("spaces/userJoined", input.PointId, updates_points.Join{PointId: input.PointId, User: user}, true, []string{})
	}, false)
	return outputsinvites.AcceptOutput{}, nil
}

// Decline /invites/decline check [ true false false ] access [ true false false false POST ]
func (a *Actions) Decline(state state.IState, input inputsinvites.DeclineInput) (any, error) {
	trx := state.Trx()
	if trx.GetLink("invite::"+input.PointId+"::"+state.Info().UserId()) != "true" {
		return nil, errors.New("invitation does not exist")
	}
	trx.DelKey("link::invite::" + input.PointId + "::" + state.Info().UserId())
	admins, err := trx.GetLinksList("admin::"+input.PointId+"::", -1, -1)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	for _, admin := range admins {
		future.Async(func() {
			a.app.Tools().Signaler().SignalUser("invites/decline", "", admin, updatesinvites.Accept{UserId: state.Info().UserId(), PointId: input.PointId}, true)
		}, false)
	}
	return outputsinvites.DeclineOutput{}, nil
}
