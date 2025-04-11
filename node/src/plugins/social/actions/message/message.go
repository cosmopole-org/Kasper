package social_services

import (
	"encoding/json"
	"errors"
	"kasper/src/abstract"
	game_model "kasper/src/plugins/game/model"
	inputs_message "kasper/src/plugins/social/inputs/message"
	models "kasper/src/plugins/social/model"
	outputs_message "kasper/src/plugins/social/outputs/message"
	"kasper/src/shell/api/model"
	"kasper/src/shell/layer1/adapters"
	module_state "kasper/src/shell/layer1/module/state"
	module_model "kasper/src/shell/layer2/model"
	"kasper/src/shell/utils/crypto"
	"log"
	"sort"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	s.AutoMigrate(&models.Seen{})
	s.AutoMigrate(&models.Message{})
	return nil
}

// CreateMessage /messages/create check [ true true true ] access [ true false false false PUT ]
func (a *Actions) CreateMessage(s abstract.IState, input inputs_message.CreateMessageInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	trx := state.Trx()
	if state.Info().TopicId() == ("main@" + a.Layer.Core().Id()) {
		type sender struct {
			Id   string         `json:"id"`
			Data datatypes.JSON `json:"data"`
		}
		meta := game_model.Meta{Id: "hokm"}
		trx.Db().First(&meta)
		if cdRaw, ok := meta.Data["chatDisabled"]; ok {
			if cd, ok2 := cdRaw.(bool); ok2 {
				if cd {
					return nil, errors.New("chat is disabled by admin")
				}
			}
		}
		trx.ClearError()

		senderUser := sender{}
		trx.Db().Model(&model.User{}).Select("id, "+adapters.BuildJsonFetcher("metadata", "hokm")+" as data").Where("id = ?", state.Info().UserId()).First(&senderUser)
		str, convErr := json.Marshal(senderUser.Data)
		if convErr != nil {
			log.Println(convErr)
		}
		dict := map[string]any{}
		convErr2 := json.Unmarshal(str, &dict)
		if convErr2 != nil {
			log.Println(convErr2)
		}
		chatBannedRaw, ok3 := dict["chatBanned"]
		if ok3 {
			chatBanned, ok := chatBannedRaw.(bool)
			if ok && chatBanned {
				return nil, errors.New("you are banned from chat")
			}
		}
		lastBuy, ok := dict["lastChatBuy"]
		chatPoint, ok2 := dict["chat"]
		if !ok || !ok2 {
			log.Println("field not found")
		}
		if (float64(time.Now().UnixMilli()) - lastBuy.(float64)) > (chatPoint.(float64) * 24 * 60 * 60 * 1000) {
			return nil, errors.New("not enough chat points")
		}
		trx.ClearError()
	}
	typ := ""
	if state.Info().IsGod() {
		typ = "adminMessage"
	} else {
		typ = "textMessage"
	}
	message := models.Message{
		Id:       crypto.SecureUniqueId(a.Layer.Core().Id()),
		AuthorId: state.Info().UserId(),
		MemberId: state.Info().MemberId(),
		SpaceId:  state.Info().SpaceId(),
		TopicId:  state.Info().TopicId(),
		Time:     time.Now().UnixMilli(),
		Data:     models.Json(input.Data),
		Typ:      typ,
	}
	space := model.Space{Id: state.Info().SpaceId()}
	trx.Db().First(&space)
	if !strings.HasPrefix(space.Tag, "match_") {
		err := trx.Db().Create(&message).Error
		if err != nil {
			return nil, err
		}
	}

	tb := abstract.UseToolbox[*module_model.ToolboxL2](a.Layer.Tools())

	type author struct {
		Id      string         `json:"id"`
		Profile datatypes.JSON `json:"profile"`
	}
	authorUser := author{}
	trx.Db().Model(&model.User{}).Select("id, "+adapters.BuildJsonFetcher("metadata", "hokm.profile")+" as profile").Where("id = ?", message.AuthorId).First(&authorUser)
	result := models.ResultMessage{
		Id:       message.Id,
		SpaceId:  message.SpaceId,
		TopicId:  message.TopicId,
		AuthorId: message.AuthorId,
		MemberId: message.MemberId,
		Data:     message.Data,
		Time:     message.Time,
		Author:   authorUser.Profile,
		Typ:      message.Typ,
	}
	tb.Signaler().SignalGroup("/messages/create", state.Info().SpaceId(), result, true, []string{state.Info().UserId()})
	return outputs_message.CreateMessageOutput{Message: message}, nil
}

// UpdateMessage /messages/update check [ true true true ] access [ true false false false PUT ]
func (a *Actions) UpdateMessage(s abstract.IState, input inputs_message.UpdateMessageInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	trx := state.Trx()
	message := models.Message{
		Id: input.MessageId,
	}
	err := trx.Db().First(&message).Error
	if err != nil {
		return nil, err
	}
	if message.AuthorId != state.Info().UserId() {
		return nil, errors.New("access to message denied")
	}
	message.Data = models.Json(input.Data)
	err2 := trx.Db().Save(&message).Error
	if err2 != nil {
		return nil, err2
	}
	tb := abstract.UseToolbox[*module_model.ToolboxL2](a.Layer.Tools())
	tb.Signaler().SignalGroup("/messages/update", state.Info().SpaceId(), message, true, []string{state.Info().UserId()})
	return outputs_message.UpdateMessageOutput{}, nil
}

// DeleteMessage /messages/delete check [ true true true ] access [ true false false false PUT ]
func (a *Actions) DeleteMessage(s abstract.IState, input inputs_message.DeleteMessageInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	trx := state.Trx()
	message := models.Message{
		Id: input.MessageId,
	}
	err1 := trx.Db().First(&message).Error
	if err1 != nil {
		return nil, err1
	}
	if message.AuthorId != state.Info().UserId() {
		return nil, errors.New("access to message denied")
	}
	err2 := trx.Db().Delete(&message).Error
	if err2 != nil {
		return nil, err2
	}
	tb := abstract.UseToolbox[*module_model.ToolboxL2](a.Layer.Tools())
	tb.Signaler().SignalGroup("/messages/delete", state.Info().SpaceId(), message, true, []string{state.Info().UserId()})
	return outputs_message.DeleteMessageOutput{}, nil
}

// SeeChat /messages/seeChat check [ true true true ] access [ true false false false PUT ]
func (a *Actions) SeeChat(s abstract.IState, input inputs_message.SeeChatInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	trx := state.Trx()
	lastMsgId := ""
	lastMsg := models.Message{}
	trx.Db().Model(&models.Message{}).Where("topic_id = ?", state.Info().TopicId()).Last(&lastMsg)
	trx.ClearError()
	lastMsgId = lastMsg.Id
	seen := models.Seen{Id: state.Info().TopicId() + "_" + state.Info().UserId(), ChatId: state.Info().TopicId(), LastMsgId: lastMsgId, UserId: state.Info().UserId()}
	trx.Db().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"last_msg_id": lastMsgId}),
	}).Create(&seen)
	trx.ClearError()
	return map[string]any{}, nil
}

type Chat struct {
	UserId      string            `json:"userId"`
	IsOnline    bool              `json:"isOnline"`
	Profile     datatypes.JSON    `json:"profile"`
	LastMsgTime int64             `json:"lastMsgTime"`
	Interaction model.Interaction `json:"interaction"`
}

// ReadChats /messages/readChats check [ true false false ] access [ true false false false PUT ]
func (a *Actions) ReadChats(s abstract.IState, input inputs_message.ReadChatsInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	toolbox := abstract.UseToolbox[*module_model.ToolboxL2](a.Layer.Tools())
	trx := state.Trx()
	var interactions = []model.Interaction{}
	err := trx.Db().Select("*").Where("user_ids LIKE ?", state.Info().UserId()+"|%").Or("user_ids LIKE ?", "%|"+state.Info().UserId()+"::"+a.Layer.Core().Id()).Or("user_ids LIKE ?", "%|"+state.Info().UserId()+"::global").Or("user_ids LIKE ?", "%|"+state.Info().UserId()).Find(&interactions).Error
	if err != nil {
		log.Println(err)
		return nil, err
	}
	chats := []*Chat{}
	authorIds := []string{}
	type author struct {
		Id      string
		Profile datatypes.JSON
	}
	for _, interaction := range interactions {
		ti := interaction.State["topicId"]
		if ti == nil {
			continue
		}
		topicId := ti.(string)
		if interaction.State["areFriends"] == "true" {
			var participantId string
			var userIds = interaction.UserIds
			if strings.HasSuffix(userIds, "::"+a.Layer.Core().Id()) {
				userIds = strings.Split(userIds, "::")[0]
			}
			var idPair = strings.Split(userIds, "|")
			if idPair[0] == state.Info().UserId() {
				participantId = idPair[1]
			} else {
				participantId = idPair[0]
			}
			lastMsg := models.Message{}
			trx.Db().Model(&models.Message{}).Where("topic_id = ?", topicId).Last(&lastMsg)
			trx.ClearError()
			if lastMsg.Time > 0 {
				authorIds = append(authorIds, participantId)
				var isOnline = toolbox.Signaler().Listeners.Has(participantId)
				chats = append(chats, &Chat{UserId: participantId, LastMsgTime: lastMsg.Time, Interaction: interaction, IsOnline: isOnline})
			}
		}
	}
	authors := []author{}
	trx.Db().Model(&model.User{}).Select("id as id, "+adapters.BuildJsonFetcher("metadata", "hokm.profile")+" as profile").Where("id in ?", authorIds).Find(&authors)
	authorDict := map[string]author{}
	for _, a := range authors {
		authorDict[a.Id] = a
	}
	for _, chat := range chats {
		chat.Profile = authorDict[chat.UserId].Profile
	}
	sort.Slice(chats, func(i, j int) bool {
		return chats[i].LastMsgTime > chats[j].LastMsgTime
	})
	return map[string]any{"chats": chats}, nil
}

// ReadMessages /messages/read check [ true true true ] access [ true false false false PUT ]
func (a *Actions) ReadMessages(s abstract.IState, input inputs_message.ReadMessagesInput) (any, error) {
	state := abstract.UseState[module_state.IStateL1](s)
	messages := []models.Message{}
	trx := state.Trx()
	trx.Db().Where("topic_id = ?", state.Info().TopicId()).Order("time desc").Offset(*input.Offset).Limit(*input.Count).Find(&messages)
	authorIds := []string{}
	for _, msg := range messages {
		authorIds = append(authorIds, msg.AuthorId)
	}
	type author struct {
		Id      string
		Profile datatypes.JSON
	}
	authors := []author{}
	trx.Db().Model(&model.User{}).Select("id as id, "+adapters.BuildJsonFetcher("metadata", "hokm.profile")+" as profile").Where("id in ?", authorIds).Find(&authors)
	authorDict := map[string]author{}
	for _, a := range authors {
		authorDict[a.Id] = a
	}
	result := []models.ResultMessage{}
	for _, msg := range messages {
		result = append(result, models.ResultMessage{
			Id:       msg.Id,
			SpaceId:  msg.SpaceId,
			TopicId:  msg.TopicId,
			AuthorId: msg.AuthorId,
			MemberId: msg.MemberId,
			Data:     msg.Data,
			Time:     msg.Time,
			Typ:      msg.Typ,
			Author:   authorDict[msg.AuthorId].Profile,
		})
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return outputs_message.ReadMessagesOutput{Messages: result}, nil
}
