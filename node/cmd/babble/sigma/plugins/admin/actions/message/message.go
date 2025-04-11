package actions_message

import (
	"errors"
	"kasper/cmd/babble/sigma/abstract"
	model "kasper/cmd/babble/sigma/api/model"
	"kasper/cmd/babble/sigma/layer1/adapters"
	module_state "kasper/cmd/babble/sigma/layer1/module/state"
	toolboxl1 "kasper/cmd/babble/sigma/layer1/module/toolbox"
	admin_inputs_message "kasper/cmd/babble/sigma/plugins/admin/inputs/message"
	admin_model "kasper/cmd/babble/sigma/plugins/admin/model"
	admin_outputs_message "kasper/cmd/babble/sigma/plugins/admin/outputs/message"
	models "kasper/cmd/babble/sigma/plugins/social/model"
	"kasper/cmd/babble/sigma/utils/crypto"

	"gorm.io/gorm"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return s.AutoMigrate(&models.Message{})
}

// SwitchChatBanned /admin/messages/switchChatBanned check [ true false false ] access [ true false false false POST ]
func (a *Actions) SwitchChatBanned(s abstract.IState, input admin_inputs_message.BanChatInput) (any, error) {
	var state = abstract.UseState[module_state.IStateL1](s)
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	trx := state.Trx()
	err := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", input.UserId) }, &model.User{Id: input.UserId}, "metadata", input.GameKey+".chatBanned", input.Banned)
	if err != nil {
		return nil, err
	}
	return map[string]any{}, nil
}

// GrantChatPerm /admin/messages/grant check [ true true true ] access [ true false false false PUT ]
func (a *Actions) GrantChatPerm(s abstract.IState, input admin_inputs_message.GrantChatInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	trx := state.Trx()
	perm := admin_model.ChatPerm{}
	err1 := trx.Db().Model(&admin_model.ChatPerm{}).Where("user_id = ?", input.UserId).First(&perm).Error
	if err1 != nil {
		perm = admin_model.ChatPerm{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), UserId: input.UserId, Time: input.Time}
		trx.Db().Create(&perm)
		return admin_outputs_message.GrantChatOutput{}, nil
	}
	perm.Time = input.Time
	trx.Db().Save(&perm)
	return admin_outputs_message.GrantChatOutput{}, nil
}

// DeleteMessage /admin/messages/delete check [ true true true ] access [ true false false false PUT ]
func (a *Actions) DeleteMessage(s abstract.IState, input admin_inputs_message.DeleteMessageInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	trx := state.Trx()
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	message := models.Message{
		Id: input.MessageId,
	}
	err2 := trx.Db().Delete(&message).Error
	if err2 != nil {
		return nil, err2
	}
	tb := abstract.UseToolbox[toolboxl1.IToolboxL1](a.Layer.Tools())
	tb.Signaler().SignalGroup("/messages/delete", state.Info().SpaceId(), message, true, []string{state.Info().UserId()})

	return admin_outputs_message.DeleteMessageOutput{}, nil
}

// ClearMessages /admin/messages/clear check [ true false false ] access [ true false false false PUT ]
func (a *Actions) ClearMessages(s abstract.IState, input admin_inputs_message.ClearMessagesInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	if !state.Info().IsGod() {
		return nil, errors.New("access denied")
	}
	trx := state.Trx()
	trx.Db().Where("topic_id = ?", input.TopicId).Delete(&admin_model.Message{})

	return map[string]any{}, nil
}
