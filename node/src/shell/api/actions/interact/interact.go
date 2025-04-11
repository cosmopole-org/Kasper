package actions_interact

import (
	"encoding/json"
	"errors"
	"fmt"
	"kasper/src/abstract"
	moduleactormodel "kasper/src/core/module/actor/model"
	inputs_interact "kasper/src/shell/api/inputs/interact"
	inputs_spaces "kasper/src/shell/api/inputs/spaces"
	inputs_users "kasper/src/shell/api/inputs/users"
	model "kasper/src/shell/api/model"
	outputs_interact "kasper/src/shell/api/outputs/interact"
	outputs_spaces "kasper/src/shell/api/outputs/spaces"
	outputs_users "kasper/src/shell/api/outputs/users"
	"kasper/src/shell/layer1/adapters"
	modulestate "kasper/src/shell/layer1/module/state"
	moduletoolbox "kasper/src/shell/layer1/module/toolbox"
	"log"
	"sort"
	"strings"

	"gorm.io/gorm/clause"
)

const userIdsLike = "user_ids LIKE ?"
const requestTemplate = "%s::requested::%s"
const blockTemplate = "%s::blocked::%s"
const blockedByTheUser = "you are blocked by the user"

func (a *Actions) getMergedUserIds(userId1 string, userId2 string, origin string) string {
	key := ""
	if userId1 > userId2 {
		if origin == "" {
			key = userId1 + "|" + userId2 + "::" + a.Layer.Core().Id()
		} else {
			key = userId1 + "|" + userId2 + "::" + origin
		}
	} else {
		if origin == "" {
			key = userId2 + "|" + userId1 + "::" + a.Layer.Core().Id()
		} else {
			key = userId2 + "|" + userId1 + "::" + origin
		}
	}
	return key
}

func (a *Actions) getInteractionState(trx adapters.ITrx, myUserId string, receipentId string, origin string) (model.Interaction, map[int]bool) {
	key := a.getMergedUserIds(myUserId, receipentId, origin)
	interaction := model.Interaction{UserIds: key}
	err := trx.Db().Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_ids = ?", key).First(&interaction).Error
	result := map[int]bool{}
	if interaction.State[fmt.Sprintf(blockTemplate, myUserId, receipentId)] == "true" {
		result[-2] = true
	}
	if err != nil {
		result[-1] = true
	}
	if interaction.State[fmt.Sprintf(blockTemplate, receipentId, myUserId)] == "true" {
		result[2] = true
	}
	if interaction.State[fmt.Sprintf(requestTemplate, myUserId, receipentId)] == "true" {
		result[3] = true
	}
	if interaction.State["areFriends"] == "true" {
		result[4] = true
	}
	if interaction.State["interacted"] == "true" {
		result[5] = true
	}
	trx.ClearError()
	return interaction, result
}

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return s.Db().AutoMigrate(&model.Interaction{})
}

// GetCode /interact/generateCode check [ true false false ] access [ true false false false GET ]
func (a *Actions) GetCode(s abstract.IState, _ inputs_interact.GenerateCodeDto) (any, error) {

	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()

	var res string
	err := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", "code")).Where("id = ?", state.Info().UserId()).First(&res).Error
	if err != nil {
		return nil, err
	}
	return outputs_interact.GetCodeOutput{Code: res}, nil
}

// GetInviteCode /interact/getInviteCode check [ true false false ] access [ true false false false GET ]
func (a *Actions) GetInviteCode(s abstract.IState, _ inputs_interact.GenerateCodeDto) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()

	var res string
	err := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", "code")).Where("id = ?", state.Info().UserId()).First(&res).Error
	if err != nil {
		return nil, err
	}
	return outputs_interact.GetCodeOutput{Code: "g" + res}, nil
}

// GetByCode /interact/getByCode check [ true false false ] access [ true false false false GET ]
func (a *Actions) GetByCode(s abstract.IState, input inputs_interact.GetByCodeDto) (any, error) {
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var user model.PublicUser
	trx := state.Trx()

	userId := toolbox.Cache().Get("code::" + strings.ToUpper(input.Code))
	if userId == "" {
		return nil, errors.New("user not found")
	}
	_, res, err := a.Layer.Actor().FetchAction("/users/get").Act(a.Layer.Sb().NewState(moduleactormodel.NewInfo("", "", "", ""), trx), inputs_users.GetInput{UserId: userId})
	log.Println(res, err)
	if err != nil {
		return nil, err
	}
	user = res.(outputs_users.GetOutput).User
	return outputs_users.GetOutput{User: user}, nil
}

func findSpace(interaction model.Interaction, userId string, trx adapters.ITrx) outputs_interact.InteractOutput {
	space := model.Space{Id: interaction.State["spaceId"].(string)}
	trx.Db().First(&space)
	trx.ClearError()
	topic := model.Topic{Id: interaction.State["topicId"].(string)}
	trx.Db().First(&topic)
	trx.ClearError()
	member := model.Member{}
	trx.Db().Where("user_id = ?", userId).Where("space_id = ?", space.Id).First(&member)
	return outputs_interact.InteractOutput{Interaction: interaction, Space: &space, Topic: &topic, Member: &member}
}

// Create /interact/create check [ true false false ] access [ true false false false GET ]
func (a *Actions) Create(s abstract.IState, input inputs_interact.InteractDto) (any, error) {
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	if input.UserId == state.Info().UserId() {
		return nil, errors.New("you can not interact with yourself")
	}
	userId := input.UserId
	trx := state.Trx()

	interaction, codeMap := a.getInteractionState(trx, state.Info().UserId(), userId, input.Origin())
	if len(codeMap) > 0 {
		if codeMap[2] {
			return nil, errors.New(blockedByTheUser)
		}
		if codeMap[5] {
			return findSpace(interaction, state.Info().UserId(), trx), nil
		}
	}
	if codeMap[-1] {
		interaction = model.Interaction{UserIds: a.getMergedUserIds(state.Info().UserId(), userId, input.Origin())}
		interactionState := map[string]interface{}{}
		interactionState["interacted"] = "true"
		interaction.State = interactionState
		st, err := json.Marshal(interactionState)
		if err != nil {
			return nil, err
		}
		err2 := trx.Db().Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_ids"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"state": string(st)}),
		}).Create(&interaction).Error
		if err2 != nil {
			return nil, err2
		}
		_, res, createPrivateErr := a.Layer.Actor().FetchAction("/spaces/createPrivate").Act(a.Layer.Sb().NewState(moduleactormodel.NewInfo(state.Info().UserId(), "", "", ""), trx), inputs_spaces.CreatePrivateInput{
			ParticipantId: userId,
			Orig:          input.Origin(),
		})
		if createPrivateErr != nil {
			return nil, createPrivateErr
		}
		privateRes := res.(outputs_spaces.CreateSpaceOutput)
		space := privateRes.Space
		topic := privateRes.Topic
		member := privateRes.Member
		interaction.State["spaceId"] = space.Id
		interaction.State["topicId"] = topic.Id
		toolbox.Signaler().SignalUser("interacted", "", userId, model.PreparedInteraction{UserId: userId, State: interaction.State}, true)
		return outputs_interact.InteractOutput{Interaction: interaction, Space: &space, Topic: &topic, Member: &member}, nil
	} else {
		if interaction.State["interacted"] == "true" {
			return findSpace(interaction, state.Info().UserId(), trx), nil
		}
		interaction.State["interacted"] = "true"
		err := trx.Db().Save(&interaction).Error
		if err != nil {
			return nil, err
		}
		var (
			space  model.Space
			member model.Member
			topic  model.Topic
		)
		if interaction.State["spaceId"] == nil {
			_, res, createPrivateErr := a.Layer.Actor().FetchAction("/spaces/createPrivate").Act(a.Layer.Sb().NewState(moduleactormodel.NewInfo(state.Info().UserId(), "", "", ""), trx), inputs_spaces.CreatePrivateInput{
				ParticipantId: userId,
				Orig:          input.Origin(),
			})
			if createPrivateErr != nil {
				return nil, createPrivateErr
			}
			privateRes := res.(outputs_spaces.CreateSpaceOutput)
			space = privateRes.Space
			topic = privateRes.Topic
			member = privateRes.Member
			interaction.State["spaceId"] = space.Id
			interaction.State["topicId"] = topic.Id
		}
		toolbox.Signaler().SignalUser("interacted", "", userId, model.PreparedInteraction{UserId: userId, State: interaction.State}, true)
		return outputs_interact.InteractOutput{Interaction: interaction, Space: &space, Topic: &topic, Member: &member}, nil
	}
}

// SendFriendRequest /interact/sendFriendRequest check [ true false false ] access [ true false false false GET ]
func (a *Actions) SendFriendRequest(s abstract.IState, input inputs_interact.SendFriendRequestDto) (any, error) {
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	userId := input.UserId
	if userId == "" {
		userId = toolbox.Cache().Get("code::" + strings.ToUpper(input.Code))
	}
	if userId == "" {
		return nil, errors.New("user not found")
	}
	if userId == state.Info().UserId() {
		return nil, errors.New("you can not interact with yourself")
	}
	trx := state.Trx()

	interaction, codeMap := a.getInteractionState(trx, state.Info().UserId(), userId, input.Origin())
	if len(codeMap) > 0 {
		if codeMap[2] {
			return nil, errors.New(blockedByTheUser)
		}
		if codeMap[3] {
			return nil, errors.New("you have already requested the user")
		}
		if codeMap[4] {
			return nil, errors.New("you are already friend with the user")
		}
	}
	if codeMap[-1] {
		interaction = model.Interaction{UserIds: a.getMergedUserIds(state.Info().UserId(), userId, input.Origin())}
		interactionState := map[string]interface{}{}
		interactionState[fmt.Sprintf(requestTemplate, state.Info().UserId(), userId)] = "true"
		interaction.State = interactionState
		st, err := json.Marshal(interactionState)
		if err != nil {
			return nil, err
		}
		err2 := trx.Db().Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_ids"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"state": string(st)}),
		}).Create(&interaction).Error
		if err2 != nil {
			return nil, err2
		}
	} else {
		if interaction.State["areFriends"] == "true" {
			return nil, errors.New("you are already friends")
		}
		interaction.State[fmt.Sprintf(requestTemplate, state.Info().UserId(), userId)] = "true"
		err := trx.Db().Save(&interaction).Error
		if err != nil {
			return nil, err
		}
	}
	toolbox.Signaler().SignalUser("friendReq", "", userId, model.PreparedInteraction{UserId: userId, State: interaction.State}, true)
	return outputs_interact.InteractOutput{Interaction: interaction}, nil
}

// UnfriendUser /interact/unfriendUser check [ true false false ] access [ true false false false GET ]
func (a *Actions) UnfriendUser(s abstract.IState, input inputs_interact.UnfriendUserDto) (any, error) {
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	if input.UserId == state.Info().UserId() {
		return nil, errors.New("you can not interact with yourself")
	}
	trx := state.Trx()

	interaction, codeMap := a.getInteractionState(trx, state.Info().UserId(), input.UserId, input.Origin())
	if codeMap[-1] {
		return nil, errors.New("you are not friends")
	}
	if codeMap[2] {
		if codeMap[4] {
			delete(interaction.State, "areFriends")
			err := trx.Db().Save(&interaction).Error
			if err != nil {
				return nil, err
			}
		}
		return outputs_interact.InteractOutput{Interaction: interaction}, nil
	}
	if !codeMap[4] {
		return nil, errors.New("you are not friends")
	}
	delete(interaction.State, "areFriends")
	err := trx.Db().Save(&interaction).Error
	if err != nil {
		return nil, errors.New("unfriending user failed")
	}
	toolbox.Signaler().SignalUser("unfriended", "", input.UserId, model.PreparedInteraction{UserId: input.UserId, State: interaction.State}, true)
	return outputs_interact.InteractOutput{Interaction: interaction}, nil
}

// BlockUser /interact/blockUser check [ true false false ] access [ true false false false GET ]
func (a *Actions) BlockUser(s abstract.IState, input inputs_interact.BlockDto) (any, error) {
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	if input.UserId == state.Info().UserId() {
		return nil, errors.New("you can not interact with yourself")
	}
	trx := state.Trx()

	interaction, codeMap := a.getInteractionState(trx, state.Info().UserId(), input.UserId, input.Origin())
	if codeMap[-1] {
		interaction = model.Interaction{UserIds: a.getMergedUserIds(state.Info().UserId(), input.UserId, input.Origin())}
		interactionState := map[string]interface{}{}
		interactionState[fmt.Sprintf(blockTemplate, state.Info().UserId(), input.UserId)] = "true"
		interaction.State = interactionState
		st, err := json.Marshal(interactionState)
		if err != nil {
			return nil, err
		}
		err2 := trx.Db().Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_ids"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"state": string(st)}),
		}).Create(&interaction).Error
		if err2 != nil {
			return nil, err2
		}
		return outputs_interact.InteractOutput{Interaction: interaction}, nil
	}
	if interaction.State[fmt.Sprintf(blockTemplate, state.Info().UserId(), input.UserId)] == "true" {
		return nil, errors.New("you have already blocked this user")
	}
	spaceId := interaction.State["spaceId"]
	topicId := interaction.State["topicId"]
	if spaceId != nil {
		topic := model.Topic{Id: topicId.(string)}
		trx.Db().Delete(&topic)
		space := model.Space{Id: spaceId.(string)}
		trx.Db().Delete(&space)
	}
	interaction.State[fmt.Sprintf(blockTemplate, state.Info().UserId(), input.UserId)] = "true"
	delete(interaction.State, "areFriends")
	delete(interaction.State, "spaceId")
	delete(interaction.State, "topicId")
	delete(interaction.State, fmt.Sprintf(requestTemplate, state.Info().UserId(), input.UserId))
	delete(interaction.State, fmt.Sprintf(requestTemplate, input.UserId, state.Info().UserId()))
	err := trx.Db().Save(&interaction).Error
	if err != nil {
		return nil, errors.New("unblocking user failed")
	}
	toolbox.Signaler().SignalUser("blocked", "", input.UserId, model.PreparedInteraction{UserId: input.UserId, State: interaction.State}, true)
	return outputs_interact.InteractOutput{Interaction: interaction}, nil
}

// UnblockUser /interact/unblockUser check [ true false false ] access [ true false false false GET ]
func (a *Actions) UnblockUser(s abstract.IState, input inputs_interact.BlockDto) (any, error) {
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	if input.UserId == state.Info().UserId() {
		return nil, errors.New("you can not interact with yourself")
	}
	trx := state.Trx()

	interaction, codeMap := a.getInteractionState(trx, state.Info().UserId(), input.UserId, input.Origin())
	if codeMap[-1] || !codeMap[-2] {
		return nil, errors.New("you have not blocked this user before")
	}
	delete(interaction.State, fmt.Sprintf(blockTemplate, state.Info().UserId(), input.UserId))
	err := trx.Db().Save(&interaction).Error
	if err != nil {
		return nil, errors.New("unblocking user failed")
	}
	toolbox.Signaler().SignalUser("unblocked", "", input.UserId, model.PreparedInteraction{UserId: input.UserId, State: interaction.State}, true)
	return outputs_interact.InteractOutput{Interaction: interaction}, nil
}

const friendReqNotFound = "friend request not found"

// AcceptFriendRequest /interact/acceptFriendRequest check [ true false false ] access [ true false false false GET ]
func (a *Actions) AcceptFriendRequest(s abstract.IState, input inputs_interact.BlockDto) (any, error) {
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	if input.UserId == state.Info().UserId() {
		return nil, errors.New("you can not interact with yourself")
	}
	trx := state.Trx()

	interaction, codeMap := a.getInteractionState(trx, state.Info().UserId(), input.UserId, input.Origin())
	if codeMap[2] {
		return nil, errors.New(blockedByTheUser)
	}
	if codeMap[-1] {
		return nil, errors.New(friendReqNotFound)
	}
	if interaction.State["areFriends"] == "true" {
		return nil, errors.New("you are already friends")
	}
	if interaction.State[fmt.Sprintf(requestTemplate, input.UserId, state.Info().UserId())] != "true" {
		return nil, errors.New(friendReqNotFound)
	}
	delete(interaction.State, fmt.Sprintf(requestTemplate, input.UserId, state.Info().UserId()))
	interaction.State["areFriends"] = "true"
	err := trx.Db().Save(&interaction).Error
	if err != nil {
		return nil, errors.New("accepting request failed")
	}
	toolbox.Signaler().SignalUser("acceptFriendReq", "", input.UserId, model.PreparedInteraction{UserId: input.UserId, State: interaction.State}, true)
	return outputs_interact.InteractOutput{Interaction: interaction}, nil
}

// DeclineFriendRequest /interact/declineFriendRequest check [ true false false ] access [ true false false false GET ]
func (a *Actions) DeclineFriendRequest(s abstract.IState, input inputs_interact.BlockDto) (any, error) {
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	if input.UserId == state.Info().UserId() {
		return nil, errors.New("you can not interact with yourself")
	}
	trx := state.Trx()

	interaction, codeMap := a.getInteractionState(trx, state.Info().UserId(), input.UserId, input.Origin())
	if codeMap[2] {
		return nil, errors.New(blockedByTheUser)
	}
	if codeMap[-1] || (interaction.State[fmt.Sprintf(requestTemplate, input.UserId, state.Info().UserId())] != "true") {
		return nil, errors.New(friendReqNotFound)
	}
	delete(interaction.State, fmt.Sprintf(requestTemplate, input.UserId, state.Info().UserId()))
	err := trx.Db().Save(&interaction).Error
	if err != nil {
		log.Println(err)
		return nil, errors.New("declining request failed")
	}
	toolbox.Signaler().SignalUser("declineFriendReq", "", input.UserId, model.PreparedInteraction{UserId: input.UserId, State: interaction.State}, true)
	return outputs_interact.InteractOutput{Interaction: interaction}, nil
}

// ReadBlockedList /interact/readBlockedList check [ true false false ] access [ true false false false GET ]
func (a *Actions) ReadBlockedList(s abstract.IState, input inputs_interact.ReadBlockedListDto) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	var result = []*model.PreparedInteraction{}
	trx := state.Trx()

	var interactions = []model.Interaction{}
	err := trx.Db().Select("*").Where(userIdsLike, state.Info().UserId()+"|%").Or(userIdsLike, "%|"+state.Info().UserId()+"::"+a.Layer.Core().Id()).Or(userIdsLike, "%|"+state.Info().UserId()+"::global").Or(userIdsLike, "%|"+state.Info().UserId()).Find(&interactions).Error
	if err != nil {
		return nil, err
	}
	var ids = []string{}
	var partiDict = map[string]*model.PreparedInteraction{}
	for _, interaction := range interactions {
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
		if interaction.State[fmt.Sprintf(blockTemplate, state.Info().UserId(), participantId)] == "true" {
			var isOnline = toolbox.Signaler().Listeners.Has(participantId)
			var participent = &model.PreparedInteraction{UserId: participantId, State: interaction.State, IsOnline: isOnline}
			ids = append(ids, participantId)
			partiDict[participantId] = participent
			result = append(result, participent)
		}
	}
	var users = []model.User{}
	trx.ClearError()
	trx.Db().Where("id in ?", ids).Find(&users)
	trx.ClearError()
	for _, user := range users {
		var profileStr string
		err := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", "hokm.profile")).Where("id = ?", user.Id).First(&profileStr).Error
		if err != nil {
			log.Println(err)
			trx.ClearError()
			continue
		}
		profile := map[string]any{}
		err2 := json.Unmarshal([]byte(profileStr), &profile)
		if err2 != nil {
			log.Println(err2)
		}
		partiDict[user.Id].Profile = profile
		trx.ClearError()
	}
	return outputs_interact.InteractsOutput{Interactions: result}, nil
}

// ReadFriendList /interact/readFriendList check [ true false false ] access [ true false false false GET ]
func (a *Actions) ReadFriendList(s abstract.IState, input inputs_interact.ReadBlockedListDto) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	var result = []*model.PreparedInteraction{}
	trx := state.Trx()

	var interactions = []model.Interaction{}
	err := trx.Db().Select("*").Where(userIdsLike, state.Info().UserId()+"|%").Or(userIdsLike, "%|"+state.Info().UserId()+"::"+a.Layer.Core().Id()).Or(userIdsLike, "%|"+state.Info().UserId()+"::global").Or(userIdsLike, "%|"+state.Info().UserId()).Find(&interactions).Error
	if err != nil {
		return nil, err
	}
	var ids = []string{}
	var partiDict = map[string]*model.PreparedInteraction{}
	for _, interaction := range interactions {
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
			var isOnline = toolbox.Signaler().Listeners.Has(participantId)
			var participent = &model.PreparedInteraction{UserId: participantId, State: interaction.State, IsOnline: isOnline}
			ids = append(ids, participantId)
			partiDict[participantId] = participent
			result = append(result, participent)
		}
	}
	var users = []model.User{}
	trx.ClearError()
	trx.Db().Where("id in ?", ids).Find(&users)
	trx.ClearError()
	for _, user := range users {
		var profileStr string
		err := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", "hokm.profile")).Where("id = ?", user.Id).First(&profileStr).Error
		if err != nil {
			log.Println(err)
			trx.ClearError()
			continue
		}
		profile := map[string]any{}
		err2 := json.Unmarshal([]byte(profileStr), &profile)
		if err2 != nil {
			log.Println(err2)
		}
		partiDict[user.Id].Profile = profile
		trx.ClearError()
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].IsOnline
	})
	return outputs_interact.InteractsOutput{Interactions: result}, nil
}

// ReadFriendRequestList /interact/readFriendRequestList check [ true false false ] access [ true false false false GET ]
func (a *Actions) ReadFriendRequestList(s abstract.IState, input inputs_interact.ReadBlockedListDto) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	var result = []*model.PreparedInteraction{}
	trx := state.Trx()

	var interactions = []model.Interaction{}
	err := trx.Db().Select("*").Where(userIdsLike, state.Info().UserId()+"|%").Or(userIdsLike, "%|"+state.Info().UserId()+"::"+a.Layer.Core().Id()).Or(userIdsLike, "%|"+state.Info().UserId()+"::global").Or(userIdsLike, "%|"+state.Info().UserId()).Find(&interactions).Error
	if err != nil {
		return nil, err
	}
	var ids = []string{}
	var partiDict = map[string]*model.PreparedInteraction{}
	for _, interaction := range interactions {
		var participantId string
		var userIds = interaction.UserIds
		if strings.HasSuffix(userIds, "::"+a.Layer.Core().Id()) {
			userIds = strings.Split(userIds, "::")[0]
		}
		idPair := strings.Split(userIds, "|")
		if idPair[0] == state.Info().UserId() {
			participantId = idPair[1]
		} else {
			participantId = idPair[0]
		}
		if interaction.State[fmt.Sprintf(requestTemplate, participantId, state.Info().UserId())] == "true" {
			var isOnline = toolbox.Signaler().Listeners.Has(participantId)
			var participent = &model.PreparedInteraction{UserId: participantId, State: interaction.State, IsOnline: isOnline}
			ids = append(ids, participantId)
			partiDict[participantId] = participent
			result = append(result, participent)
		}
	}
	var users = []model.User{}
	trx.ClearError()
	trx.Db().Where("id in ?", ids).Find(&users)
	trx.ClearError()
	for _, user := range users {
		var profileStr string
		err := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", "hokm.profile")).Where("id = ?", user.Id).First(&profileStr).Error
		if err != nil {
			log.Println(err)
			trx.ClearError()
			continue
		}
		profile := map[string]any{}
		err2 := json.Unmarshal([]byte(profileStr), &profile)
		if err2 != nil {
			log.Println(err2)
		}
		partiDict[user.Id].Profile = profile
		trx.ClearError()
	}
	return outputs_interact.InteractsOutput{Interactions: result}, nil
}

// ReadInteractions /interact/read check [ true false false ] access [ true false false false GET ]
func (a *Actions) ReadInteractions(s abstract.IState, input inputs_interact.ReadBlockedListDto) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	toolbox := abstract.UseToolbox[*moduletoolbox.ToolboxL1](a.Layer.Tools())
	var result = []*model.PreparedInteraction{}
	trx := state.Trx()

	var interactions = []model.Interaction{}
	err := trx.Db().Select("*").Where(userIdsLike, state.Info().UserId()+"|%").Or(userIdsLike, "%|"+state.Info().UserId()+"::"+a.Layer.Core().Id()).Or(userIdsLike, "%|"+state.Info().UserId()+"::global").Or(userIdsLike, "%|"+state.Info().UserId()).Find(&interactions).Error
	if err != nil {
		return nil, err
	}
	var ids = []string{}
	var partiDict = map[string]*model.PreparedInteraction{}
	for _, interaction := range interactions {
		if interaction.State["interacted"] == "true" {
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
			var isOnline = toolbox.Signaler().Listeners.Has(participantId)
			var participent = &model.PreparedInteraction{UserId: participantId, State: interaction.State, IsOnline: isOnline}
			ids = append(ids, participantId)
			partiDict[participantId] = participent
			result = append(result, participent)
		}
	}
	var users = []model.User{}
	trx.ClearError()
	trx.Db().Where("id in ?", ids).Find(&users)
	for _, user := range users {
		partiDict[user.Id].Profile = map[string]any{"name": user.Name, "avatar": user.Avatar}
	}
	return outputs_interact.InteractsOutput{Interactions: result}, nil
}
