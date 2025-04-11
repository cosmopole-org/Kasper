package actions_space

import (
	"errors"
	"fmt"
	"kasper/src/abstract"
	module_actor_model "kasper/src/core/module/actor/model"
	inputsspaces "kasper/src/shell/api/inputs/spaces"
	models "kasper/src/shell/api/model"
	outputsspaces "kasper/src/shell/api/outputs/spaces"
	updatesspaces "kasper/src/shell/api/updates/spaces"
	"kasper/src/shell/layer1/adapters"
	modulestate "kasper/src/shell/layer1/module/state"
	tb "kasper/src/shell/layer1/module/toolbox"
	"kasper/src/shell/utils/future"
)

const memberTemplate = "member::%s::%s::%s"

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	err := s.Db().AutoMigrate(&models.Space{})
	if err != nil {
		return err
	}
	err2 := s.Db().AutoMigrate(&models.Member{})
	if err2 != nil {
		return err2
	}
	err3 := s.Db().AutoMigrate(&models.Admin{})
	if err3 != nil {
		return err3
	}
	err4 := s.Db().AutoMigrate(&models.Topic{})
	if err4 != nil {
		return err4
	}
	space := models.Space{}
	errFind := s.Db().Where("id = ?", "main@"+a.Layer.Core().Id()).First(&space).Error
	if errFind == nil {
		var members []models.Member
		s.Db().Model(&models.Member{}).Find(&members)
		future.Async(func() {
			for _, member := range members {
				toolbox.Cache().Put(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id), member.TopicId)
				toolbox.Signaler().JoinGroup(member.SpaceId, member.UserId)
			}
		}, false)
		return nil
	}
	space = models.Space{Id: "main@" + a.Layer.Core().Id(), Tag: "main@" + a.Layer.Core().Id(), Title: "main", Avatar: "0", IsPublic: true}
	errSpace := s.Db().Create(&space).Error
	if errSpace != nil {
		return errSpace
	}
	topic := models.Topic{Id: "main@" + a.Layer.Core().Id(), Title: "hall", Avatar: "0", SpaceId: space.Id}
	errTopic := s.Db().Create(&topic).Error
	if errTopic != nil {
		return errTopic
	}
	toolbox.Cache().Put(fmt.Sprintf("city::%s", topic.Id), topic.SpaceId)

	var members []models.Member
	s.Db().Model(&models.Member{}).Find(&members)
	for _, member := range members {
		toolbox.Cache().Put(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id), member.TopicId)
		toolbox.Signaler().JoinGroup(member.SpaceId, member.UserId)
	}

	return nil
}

// AddMember /spaces/addMember check [ true true false ] access [ true false false false POST ]
func (a *Actions) AddMember(s abstract.IState, input inputsspaces.AddMemberInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var member models.Member
	trx := state.Trx()
	err := trx.Db().First(&models.User{Id: input.UserId}).Error
	if err != nil {
		return nil, err
	}
	ti := state.Info().TopicId()
	if ti == "" {
		ti = "*"
	}
	member = models.Member{Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), UserId: input.UserId, SpaceId: state.Info().SpaceId(), TopicId: ti, Metadata: input.Metadata}
	err2 := trx.Db().Create(&member).Error
	if err2 != nil {
		return nil, err2
	}
	trx.Mem().Put(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id), member.TopicId)
	toolbox.Signaler().JoinGroup(member.SpaceId, member.UserId)
	toolbox.Signaler().SignalUser("spaces/addMemberMe", "", member.UserId, updatesspaces.AddMember{SpaceId: state.Info().SpaceId(), TopicId: ti, Member: member}, true)
	future.Async(func() {
		toolbox.Signaler().SignalGroup("spaces/addMember", state.Info().SpaceId(), updatesspaces.AddMember{SpaceId: state.Info().SpaceId(), TopicId: ti, Member: member}, true, []string{state.Info().UserId()})
	}, false)
	return outputsspaces.AddMemberOutput{Member: member}, nil
}

// UpdateMember /spaces/updateMember check [ true true true ] access [ true false false false POST ]
func (a *Actions) UpdateMember(s abstract.IState, input inputsspaces.UpdateMemberInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var member = models.Member{Id: input.MemberId}
	trx := state.Trx()
	err := trx.Db().First(&member).Error
	if err != nil {
		return nil, err
	}
	if (member.TopicId != "*") && (member.TopicId == state.Info().TopicId()) {
		return nil, errors.New("access to member denied")
	}
	member.Metadata = input.Metadata
	trx.Db().Save(&member)
	future.Async(func() {
		toolbox.Signaler().SignalGroup("spaces/updateMember", state.Info().SpaceId(), updatesspaces.AddMember{SpaceId: state.Info().SpaceId(), TopicId: state.Info().TopicId(), Member: member}, true, []string{state.Info().UserId()})
	}, false)
	return outputsspaces.AddMemberOutput{Member: member}, nil
}

// ReadMembers /spaces/readMembers check [ true true false ] access [ true false false false POST ]
func (a *Actions) ReadMembers(s abstract.IState, input inputsspaces.ReadMemberInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()
	var members []models.Member
	err := trx.Db().Model(&models.Member{}).
		Where("space_id = ?", state.Info().SpaceId()).
		Find(&members).
		Error
	if err != nil {
		return nil, err
	}
	ids := []string{}
	memberDict := map[string]string{}
	for _, member := range members {
		ids = append(ids, member.UserId)
		memberDict[member.Id] = member.UserId
	}
	var users []models.User
	err2 := trx.Db().Model(&models.User{}).Where("id in ?", ids).Find(&users).Error
	if err2 != nil {
		return nil, err2
	}
	memberUsers := []outputsspaces.MemberUser{}
	userDict := map[string]models.PublicUser{}
	for _, user := range users {
		userDict[user.Id] = models.PublicUser{
			Id:        user.Id,
			Type:      user.Typ,
			Name:      user.Name,
			Avatar:    user.Avatar,
			Username:  user.Username,
			PublicKey: user.PublicKey,
		}
	}
	for _, member := range members {
		memberUsers = append(memberUsers, outputsspaces.MemberUser{User: userDict[memberDict[member.Id]], Member: member})
	}
	return outputsspaces.ReadMemberOutput{Members: memberUsers}, nil
}

// RemoveMember /spaces/removeMember check [ true true false ] access [ true false false false POST ]
func (a *Actions) RemoveMember(s abstract.IState, input inputsspaces.RemoveMemberInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var member models.Member
	trx := state.Trx()
	admin := models.Admin{}
	err := trx.Db().Where("space_id = ?", state.Info().SpaceId()).Where("user_id = ?", state.Info().UserId()).First(&admin).Error
	if err != nil {
		return nil, err
	}
	member = models.Member{Id: input.MemberId}
	err2 := trx.Db().First(&member).Error
	if err2 != nil {
		return nil, err2
	}
	ti := input.TopicId
	if ti == "" {
		ti = "*"
	}
	if ti != member.TopicId {
		return nil, errors.New("topic id does not match")
	}
	err3 := trx.Db().Delete(&member).Error
	if err3 != nil {
		return nil, err3
	}
	trx.Mem().Del(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id))
	toolbox.Signaler().LeaveGroup(member.SpaceId, state.Info().UserId())
	future.Async(func() {
		toolbox.Signaler().SignalGroup("spaces/removeMember", state.Info().SpaceId(), updatesspaces.AddMember{SpaceId: state.Info().SpaceId(), TopicId: ti, Member: member}, true, []string{state.Info().UserId()})
	}, false)
	return outputsspaces.AddMemberOutput{Member: member}, nil
}

// Create /spaces/create check [ true false false ] access [ true false false false POST ]
func (a *Actions) Create(s abstract.IState, input inputsspaces.CreateInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var (
		space  models.Space
		member models.Member
		topic  models.Topic
	)
	trx := state.Trx()
	space = models.Space{Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), Tag: input.Tag, Title: input.Title, Avatar: input.Avatar, IsPublic: input.IsPublic}
	err := trx.Db().Create(&space).Error
	if err != nil {
		return nil, err
	}
	member = models.Member{Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), UserId: state.Info().UserId(), SpaceId: space.Id, TopicId: "*", Metadata: ""}
	err2 := trx.Db().Create(&member).Error
	if err2 != nil {
		return nil, err2
	}
	admin := models.Admin{Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), UserId: state.Info().UserId(), SpaceId: space.Id, Role: "creator"}
	err3 := trx.Db().Create(&admin).Error
	if err3 != nil {
		return nil, err3
	}
	topic = models.Topic{Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), Title: "🏛️ hall", Avatar: "0", SpaceId: space.Id}
	err4 := trx.Db().Create(&topic).Error
	if err4 != nil {
		return nil, err4
	}
	trx.Mem().Put(fmt.Sprintf("city::%s", topic.Id), topic.SpaceId)
	toolbox.Signaler().JoinGroup(member.SpaceId, member.UserId)
	trx.Mem().Put(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id), member.TopicId)
	toolbox.Signaler().SignalUser("spaces/addMemberMe", "", member.UserId, updatesspaces.AddMember{SpaceId: space.Id, TopicId: topic.Id, Member: member}, true)
	return outputsspaces.CreateOutput{Space: space, Member: member, Topic: topic}, nil
}

// Update /spaces/update check [ true false false ] access [ true false false false PUT ]
func (a *Actions) Update(s abstract.IState, input inputsspaces.UpdateInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var space models.Space
	trx := state.Trx()
	admin := models.Admin{}
	err := trx.Db().Where("user_id=?", state.Info().UserId()).Where("space_id=?", input.SpaceId).First(&admin).Error
	if err != nil {
		return nil, err
	}
	space = models.Space{Id: input.SpaceId}
	err2 := trx.Db().First(&space).Error
	if err2 != nil {
		return nil, err2
	}
	space.Title = input.Title
	space.Avatar = input.Avatar
	space.Tag = input.Tag + "@" + a.Layer.Core().Id()
	space.IsPublic = input.IsPublic
	err3 := trx.Db().Save(&space).Error
	if err3 != nil {
		return nil, err3
	}
	future.Async(func() {
		toolbox.Signaler().SignalGroup("spaces/update", space.Id, updatesspaces.Update{Space: space}, true, []string{state.Info().UserId()})
	}, false)
	return outputsspaces.UpdateOutput{Space: space}, nil
}

// Delete /spaces/delete check [ true false false ] access [ true false false false DELETE ]
func (a *Actions) Delete(s abstract.IState, input inputsspaces.DeleteInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var space models.Space
	trx := state.Trx()
	admin := models.Admin{}
	err := trx.Db().Where("user_id=?", state.Info().UserId()).Where("space_id=?", input.SpaceId).First(&admin).Error
	if err != nil {
		return nil, err
	}
	if admin.Role != "creator" {
		return nil, errors.New("you are not the space creator")
	}
	space = models.Space{Id: input.SpaceId}
	err2 := trx.Db().First(&space).Error
	if err2 != nil {
		return nil, err2
	}
	err3 := trx.Db().Delete(&space).Error
	if err3 != nil {
		return nil, err3
	}
	toolbox.Signaler().LeaveGroup(space.Id, state.Info().UserId())
	future.Async(func() {
		toolbox.Signaler().SignalGroup("spaces/delete", space.Id, updatesspaces.Delete{Space: space}, true, []string{state.Info().UserId()})
	}, false)
	return outputsspaces.DeleteOutput{Space: space}, nil
}

// Get /spaces/get check [ true false false ] access [ true false false false GET ]
func (a *Actions) Get(s abstract.IState, input inputsspaces.GetInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	var space models.Space
	trx := state.Trx()
	space = models.Space{Id: input.SpaceId}
	err := trx.Db().First(&space).Error
	if err != nil {
		return nil, err
	}
	if space.IsPublic {
		return outputsspaces.GetOutput{Space: space}, nil
	}
	member := models.Member{}
	err2 := trx.Db().Where("space_id = ?", input.SpaceId).Where("user_id = ?", state.Info().UserId()).First(&member).Error
	if err2 != nil {
		return nil, err2
	}
	return outputsspaces.GetOutput{Space: space}, nil
}

// Read /spaces/read check [ true false false ] access [ true false false false GET ]
func (a *Actions) Read(s abstract.IState, input inputsspaces.ReadInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()
	members := []models.Member{}
	err := trx.Db().Model(&models.Member{}).Where("user_id = ?", state.Info().UserId()).Find(&members).Error
	if err != nil {
		return nil, err
	}
	spaceIds := []string{}
	for _, member := range members {
		spaceIds = append(spaceIds, member.SpaceId)
	}
	spaces := []models.Space{}
	err2 := trx.Db().Where("id in ?", spaceIds).Find(&spaces).Error
	if err2 != nil {
		return nil, err2
	}
	return outputsspaces.ReadOutput{Spaces: spaces}, nil
}

// Join /spaces/join check [ true false false ] access [ true false false false POST ]
func (a *Actions) Join(s abstract.IState, input inputsspaces.JoinInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var member models.Member
	trx := state.Trx()
	space := models.Space{Id: input.SpaceId}
	err := trx.Db().First(&space).Error
	if err != nil {
		return nil, err
	}
	if !space.IsPublic {
		return nil, errors.New("access to private space denied")
	}
	member = models.Member{Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), UserId: state.Info().UserId(), SpaceId: input.SpaceId, TopicId: "*", Metadata: ""}
	err2 := trx.Db().Create(&member).Error
	if err2 != nil {
		return nil, err2
	}
	toolbox.Signaler().JoinGroup(member.SpaceId, member.UserId)
	trx.Mem().Put(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id), member.TopicId)
	future.Async(func() {
		toolbox.Signaler().SignalGroup("spaces/join", member.SpaceId, updatesspaces.Join{Member: member}, true, []string{member.UserId})
	}, false)
	return outputsspaces.JoinOutput{Member: member}, nil
}

// CreateGroup /spaces/createGroup check [ true false false ] access [ true false false false PUT ]
func (a *Actions) CreateGroup(s abstract.IState, input inputsspaces.CreateGroupInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	var (
		space  models.Space
		member models.Member
		topic  models.Topic
	)
	trx := state.Trx()
	_, rawSpaceRes, spaceErr := a.Layer.Core().Get(1).Actor().FetchAction("/spaces/create").Act(a.Layer.Sb().NewState(module_actor_model.NewInfo(state.Info().UserId(), "", "", ""), trx), inputsspaces.CreateInput{
		Tag:      "group",
		Title:    input.Name,
		Avatar:   "0",
		IsPublic: false,
	})
	if spaceErr != nil {
		return nil, spaceErr
	}
	space = rawSpaceRes.(outputsspaces.CreateOutput).Space
	topic = rawSpaceRes.(outputsspaces.CreateOutput).Topic
	member = rawSpaceRes.(outputsspaces.CreateOutput).Member
	return outputsspaces.CreateSpaceOutput{Space: space, Topic: topic, Member: member}, nil
}

const requestTemplate = "%s::requested::%s"
const blockTemplate = "%s::blocked::%s"

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

func (a *Actions) getInteractionState(trx adapters.ITrx, myUserId string, receipentId string, origin string) (models.Interaction, map[int]bool) {
	key := a.getMergedUserIds(myUserId, receipentId, origin)
	interaction := models.Interaction{UserIds: key}
	err := trx.Db().Where("user_ids = ?", key).First(&interaction).Error
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
	return interaction, result
}

// CreatePrivate /spaces/createPrivate check [ true false false ] access [ true false false false PUT ]
func (a *Actions) CreatePrivate(s abstract.IState, input inputsspaces.CreatePrivateInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	var (
		myMember models.Member
		space    models.Space
		topic    models.Topic
	)
	trx := state.Trx()
	interaction, codeMap := a.getInteractionState(trx, state.Info().UserId(), input.ParticipantId, input.Origin())
	if codeMap[-1] {
		return nil, errors.New("interaction not found")
	}
	if codeMap[2] {
		return nil, errors.New("you are blocked by the user")
	}
	if !codeMap[4] && !codeMap[5] {
		return nil, errors.New("you 2 are not friends and not even interacted")
	}
	if interaction.State["spaceId"] != nil {
		return nil, errors.New("private room already exists")
	}
	_, res, spaceErr := a.Layer.Actor().FetchAction("/spaces/create").Act(a.Layer.Sb().NewState(module_actor_model.NewInfo(state.Info().UserId(), myMember.SpaceId, "", myMember.Id), trx), inputsspaces.CreateInput{
		Tag:      "private",
		Title:    "private_space",
		Avatar:   "0",
		IsPublic: false,
	})
	if spaceErr != nil {
		return nil, spaceErr
	}
	groupRes := res.(outputsspaces.CreateOutput)
	space = groupRes.Space
	myMember = groupRes.Member
	topic = groupRes.Topic
	_, _, addMemberErr := a.Layer.Actor().FetchAction("/spaces/addMember").Act(a.Layer.Sb().NewState(module_actor_model.NewInfo(state.Info().UserId(), myMember.SpaceId, "", myMember.Id), trx), inputsspaces.AddMemberInput{
		UserId:   input.ParticipantId,
		SpaceId:  groupRes.Space.Id,
		Metadata: "",
	})
	if addMemberErr != nil {
		return nil, addMemberErr
	}
	interaction.State["spaceId"] = space.Id
	interaction.State["topicId"] = topic.Id
	err2 := trx.Db().Save(&interaction).Error
	if err2 != nil {
		return nil, err2
	}
	return outputsspaces.CreateSpaceOutput{Space: space, Member: myMember, Topic: topic}, nil
}
