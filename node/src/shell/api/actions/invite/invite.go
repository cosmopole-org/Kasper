package actions_invite

import (
	"errors"
	"fmt"
	"kasper/src/abstract"
	inputsinvites "kasper/src/shell/api/inputs/invites"
	"kasper/src/shell/api/model"
	outputsinvites "kasper/src/shell/api/outputs/invites"
	updatesinvites "kasper/src/shell/api/updates/invites"
	"kasper/src/shell/layer1/adapters"
	modulestate "kasper/src/shell/layer1/module/state"
	toolbox2 "kasper/src/shell/layer1/module/toolbox"
	"kasper/src/shell/utils/future"
)

const inviteNotFoundError = "invite not found"

var memberTemplate = "member::%s::%s::%s"

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return s.Db().AutoMigrate(&model.Invite{})
}

// Create /invites/create check [ true true false ] access [ true false false false POST ]
func (a *Actions) Create(s abstract.IState, input inputsinvites.CreateInput) (any, error) {
	toolbox := abstract.UseToolbox[*toolbox2.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var invite model.Invite
	trx := state.Trx()
	space := model.Space{Id: input.SpaceId}
	err := trx.Db().First(&space).Error
	if err != nil {
		return nil, err
	}
	invite = model.Invite{Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), UserId: input.UserId, SpaceId: input.SpaceId}
	err2 := trx.Db().Create(&invite).Error
	if err2 != nil {
		return nil, err2
	}
	future.Async(func() {
		toolbox.Signaler().SignalUser("invites/create", "", input.UserId, updatesinvites.Create{Invite: invite}, true)
	}, false)
	return outputsinvites.CreateOutput{Invite: invite}, nil
}

// Cancel /invites/cancel check [ true true false ] access [ true false false false POST ]
func (a *Actions) Cancel(s abstract.IState, input inputsinvites.CancelInput) (any, error) {
	toolbox := abstract.UseToolbox[*toolbox2.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var invite model.Invite
	trx := state.Trx()
	admin := model.Admin{UserId: state.Info().UserId(), SpaceId: input.SpaceId}
	err := trx.Db().First(&admin).Error
	if err != nil {
		return nil, err
	}
	invite = model.Invite{Id: input.InviteId}
	err2 := trx.Db().First(&invite).Error
	if err2 != nil {
		return nil, err2
	}
	if invite.SpaceId != input.SpaceId {
		return nil, errors.New(inviteNotFoundError)
	}
	err3 := trx.Db().Delete(&invite).Error
	if err3 != nil {
		return nil, err3
	}
	future.Async(func() {
		toolbox.Signaler().SignalUser("invites/cancel", "", invite.UserId, updatesinvites.Cancel{Invite: invite}, true)
	}, false)
	return outputsinvites.CancelOutput{Invite: invite}, nil
}

// Accept /invites/accept check [ true false false ] access [ true false false false POST ]
func (a *Actions) Accept(s abstract.IState, input inputsinvites.AcceptInput) (any, error) {
	toolbox := abstract.UseToolbox[*toolbox2.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var member model.Member
	trx := state.Trx()
	invite := model.Invite{Id: input.InviteId}
	err := trx.Db().First(&invite).Error
	if err != nil {
		return nil, err
	}
	if invite.UserId != state.Info().UserId() {
		return nil, errors.New(inviteNotFoundError)
	}
	err2 := trx.Db().Delete(&invite).Error
	if err2 != nil {
		return nil, err2
	}
	member = model.Member{Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), UserId: invite.UserId, SpaceId: invite.SpaceId, TopicId: "*", Metadata: ""}
	err3 := trx.Db().Create(&member).Error
	if err3 != nil {
		return nil, err3
	}
	toolbox.Signaler().JoinGroup(member.SpaceId, member.UserId)
	trx.Mem().Put(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id), member.TopicId)
	var admins []model.Admin
	err4 := trx.Db().Where("space_id = ?", invite.SpaceId).Find(&admins).Error
	if err4 != nil {
		return nil, err4
	}
	for _, admin := range admins {
		future.Async(func() {
			toolbox.Signaler().SignalUser("invites/accept", "", admin.UserId, updatesinvites.Accept{Invite: invite}, true)
		}, false)
	}
	future.Async(func() {
		toolbox.Signaler().SignalGroup("spaces/userJoined", invite.SpaceId, updatesinvites.Accept{Invite: invite}, true, []string{})
	}, false)
	return outputsinvites.AcceptOutput{Member: member}, nil
}

// Decline /invites/decline check [ true false false ] access [ true false false false POST ]
func (a *Actions) Decline(s abstract.IState, input inputsinvites.DeclineInput) (any, error) {
	toolbox := abstract.UseToolbox[*toolbox2.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()
	invite := model.Invite{Id: input.InviteId}
	err := trx.Db().First(&invite).Error
	if err != nil {
		return nil, err
	}
	if invite.UserId != state.Info().UserId() {
		return nil, errors.New(inviteNotFoundError)
	}
	err2 := trx.Db().Delete(&invite).Error
	if err2 != nil {
		return nil, err2
	}
	var admins []model.Admin
	err3 := trx.Db().Where("space_id = ?", invite.SpaceId).Find(&admins).Error
	if err3 != nil {
		return nil, err3
	}
	for _, admin := range admins {
		future.Async(func() {
			toolbox.Signaler().SignalUser("invites/decline", "", admin.UserId, updatesinvites.Accept{Invite: invite}, true)
		}, false)
	}
	return outputsinvites.DeclineOutput{}, nil
}
