package actions_topic

import (
	"errors"
	"fmt"
	"kasper/src/abstract"
	inputstopics "kasper/src/shell/api/inputs/topics"
	"kasper/src/shell/api/model"
	outputstopics "kasper/src/shell/api/outputs/topics"
	updatestopics "kasper/src/shell/api/updates/topics"
	"kasper/src/shell/layer1/adapters"
	modulestate "kasper/src/shell/layer1/module/state"
	tb "kasper/src/shell/layer1/module/toolbox"
	"kasper/src/shell/utils/future"
	"log"
)

var sendTemplate = "topics/send"

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	err := s.Db().AutoMigrate(&model.Topic{})
	if err != nil {
		return err
	}
	return nil
}

// Create /topics/create check [ true true false ] access [ true false false false POST ]
func (a *Actions) Create(s abstract.IState, input inputstopics.CreateInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var topic model.Topic
	trx := state.Trx()
	space := model.Space{Id: state.Info().SpaceId()}
	err := trx.Db().First(&space).Error
	if err != nil {
		return nil, err
	}
	topic = model.Topic{Id: toolbox.Cache().GenId(trx.Db(), input.Origin()), Title: input.Title, Avatar: input.Avatar, SpaceId: space.Id}
	err2 := trx.Db().Create(&topic).Error
	if err2 != nil {
		return nil, err2
	}
	trx.Mem().Put(fmt.Sprintf("city::%s", topic.Id), topic.SpaceId)
	future.Async(func() {
		toolbox.Signaler().SignalGroup("topics/create", topic.SpaceId, updatestopics.Create{Topic: topic}, true, []string{state.Info().UserId()})
	}, false)
	return outputstopics.CreateOutput{Topic: topic}, nil
}

// Update /topics/update check [ true true false ] access [ true false false false PUT ]
func (a *Actions) Update(s abstract.IState, input inputstopics.UpdateInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var topic model.Topic
	trx := state.Trx()
	space := model.Space{Id: state.Info().SpaceId()}
	err := trx.Db().First(&space).Error
	if err != nil {
		return nil, err
	}
	topic = model.Topic{Id: input.TopicId}
	err2 := trx.Db().First(&topic).Error
	if err2 != nil {
		return nil, err2
	}
	topic.Title = input.Title
	topic.Avatar = input.Avatar
	err3 := trx.Db().Save(&topic).Error
	if err3 != nil {
		return nil, err3
	}
	future.Async(func() {
		toolbox.Signaler().SignalGroup("topics/update", topic.SpaceId, updatestopics.Update{Topic: topic}, true, []string{state.Info().UserId()})
	}, false)
	return outputstopics.UpdateOutput{Topic: topic}, nil
}

// Delete /topics/delete check [ true true false ] access [ true false false false DELETE ]
func (a *Actions) Delete(s abstract.IState, input inputstopics.DeleteInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	var topic model.Topic
	trx := state.Trx()
	space := model.Space{Id: state.Info().SpaceId()}
	err := trx.Db().First(&space).Error
	if err != nil {
		return nil, err
	}
	topic = model.Topic{Id: input.TopicId}
	err2 := trx.Db().First(&topic).Error
	if err2 != nil {
		return nil, err2
	}
	err3 := trx.Db().Delete(&topic).Error
	if err3 != nil {
		return nil, err3
	}
	trx.Mem().Del(fmt.Sprintf("city::%s", topic.Id))
	future.Async(func() {
		toolbox.Signaler().SignalGroup("topics/delete", topic.SpaceId, updatestopics.Delete{Topic: topic}, true, []string{state.Info().UserId()})
	}, false)
	return outputstopics.DeleteOutput{Topic: topic}, nil
}

// Get /topics/get check [ true true false ] access [ true false false false GET ]
func (a *Actions) Get(s abstract.IState, input inputstopics.GetInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	var topic model.Topic
	trx := state.Trx()
	space := model.Space{Id: state.Info().SpaceId()}
	err := trx.Db().First(&space).Error
	if err != nil {
		return nil, err
	}
	topic = model.Topic{Id: input.TopicId}
	err2 := trx.Db().First(&topic).Error
	if err2 != nil {
		return nil, err2
	}
	if space.IsPublic {
		return outputstopics.GetOutput{Topic: topic}, nil
	}
	member := model.Member{}
	err3 := trx.Db().Where("space_id = ?", space.Id).Where("user_id = ?", state.Info().UserId()).First(&member).Error
	if err3 != nil {
		return nil, errors.New("access to space denied")
	}
	return outputstopics.GetOutput{Topic: topic}, nil
}

// Read /topics/read check [ true false false ] access [ true false false false GET ]
func (a *Actions) Read(s abstract.IState, input inputstopics.ReadInput) (any, error) {
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()
	space := model.Space{Id: input.SpaceId}
	err := trx.Db().First(&space).Error
	if err != nil {
		return nil, err
	}
	topics := []model.Topic{}
	err2 := trx.Db().Where("space_id = ?", space.Id).Find(&topics).Error
	if err2 != nil {
		return nil, err2
	}
	if space.IsPublic {
		return outputstopics.ReadOutput{Topics: topics}, nil
	}
	member := model.Member{}
	err3 := trx.Db().Where("space_id = ?", space.Id).Where("user_id = ?", state.Info().UserId()).First(&member).Error
	if err3 != nil {
		return nil, errors.New("access to space denied")
	}
	return outputstopics.ReadOutput{Topics: topics}, nil
}

// Send /topics/send check [ true true true ] access [ true false false false POST ]
func (a *Actions) Send(s abstract.IState, input inputstopics.SendInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	state := abstract.UseState[modulestate.IStateL1](s)
	trx := state.Trx()
	member := model.Member{Id: input.MemberId}
	errMember := trx.Db().First(&member).Error
	if errMember != nil {
		log.Println(errMember)
		trx.ClearError()
	}
	if member.UserId == state.Info().UserId() && member.SpaceId == state.Info().SpaceId() {
		if input.Type == "broadcast" {
			user := model.User{Id: state.Info().UserId()}
			trx.Db().Select("id as id, name as name, avatar as avatar").First(&user)
			trx.ClearError()
			var p = updatestopics.Send{Action: "broadcast", Member: member, User: user, Topic: model.Topic{SpaceId: state.Info().SpaceId(), Id: input.TopicId}, Data: input.Data}
			toolbox.Signaler().SignalGroup(sendTemplate, state.Info().SpaceId(), p, true, []string{state.Info().UserId()})
			return outputstopics.SendOutput{Passed: true}, nil
		} else if input.Type == "single" {
			targetMember := model.Member{Id: input.RecvId}
			err := trx.Db().First(&targetMember).Error
			if err != nil {
				log.Println(err)
				trx.ClearError()
			}
			if targetMember.SpaceId == state.Info().SpaceId() {
				user := model.User{Id: state.Info().UserId()}
				trx.Db().Select("id as id, name as name, avatar as avatar").First(&user)
				var p = updatestopics.Send{Action: "single", User: user, Member: member, TargetMember: targetMember, Topic: model.Topic{SpaceId: state.Info().SpaceId(), Id: input.TopicId}, Data: input.Data}
				toolbox.Signaler().SignalUser(sendTemplate, "", targetMember.UserId, p, true)
				return outputstopics.SendOutput{Passed: true}, nil
			}
		}
	}
	return outputstopics.SendOutput{Passed: false}, nil
}
