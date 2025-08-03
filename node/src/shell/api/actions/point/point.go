package actions_space

import (
	"errors"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/state"
	inputs_points "kasper/src/shell/api/inputs/points"
	"kasper/src/shell/api/model"
	outputs_points "kasper/src/shell/api/outputs/points"
	updates_points "kasper/src/shell/api/updates/points"
	"kasper/src/shell/utils/future"
	"log"
	"strings"
	"time"
)

type Actions struct {
	App core.ICore
}

func Install(a *Actions) error {
	return nil
}

// AddMember /points/addMember check [ true true false ] access [ true false false false POST ]
func (a *Actions) AddMember(state state.IState, input inputs_points.AddMemberInput) (any, error) {
	trx := state.Trx()
	if !trx.HasObj("User", input.UserId) {
		return nil, errors.New("user not found")
	}
	trx.PutLink("member::"+state.Info().PointId()+"::"+input.UserId, "true")
	trx.PutLink("memberof::"+input.UserId+"::"+state.Info().PointId(), "true")
	a.App.Tools().Signaler().JoinGroup(state.Info().PointId(), input.UserId)
	user := model.User{Id: input.UserId}.Pull(trx)
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/addMember", state.Info().PointId(), updates_points.AddMember{PointId: state.Info().PointId(), User: user}, true, []string{state.Info().UserId()})
	}, false)
	return outputs_points.AddMemberOutput{}, nil
}

// UpdateMember /points/updateMember check [ true true true ] access [ true false false false POST ]
func (a *Actions) UpdateMember(state state.IState, input inputs_points.UpdateMemberInput) (any, error) {
	trx := state.Trx()
	if state.Info().PointId() == "" {
		return nil, errors.New("member not found")
	}
	trx.PutJson("member_"+state.Info().PointId()+"_"+input.UserId, "meta", input.Metadata, true)
	user := model.User{Id: input.UserId}.Pull(trx)
	obj, e := trx.GetJson("member_"+state.Info().PointId()+"_"+input.UserId, "meta")
	if e == nil {
		future.Async(func() {
			a.App.Tools().Signaler().SignalGroup("points/updateMember", state.Info().PointId(), updates_points.UpdateMember{PointId: state.Info().PointId(), User: user, Metadata: obj}, true, []string{state.Info().UserId()})
		}, false)
	}
	return outputs_points.UpdateMemberOutput{Metadata: obj}, nil
}

// ReadMembers /points/readMembers check [ true true false ] access [ true false false false POST ]
func (a *Actions) ReadMembers(state state.IState, input inputs_points.ReadMemberInput) (any, error) {
	trx := state.Trx()
	members, err := model.User{}.List(trx, "member::"+state.Info().PointId()+"::")
	if err != nil {
		return nil, err
	}
	return outputs_points.ReadMemberOutput{Members: members}, nil
}

// RemoveMember /points/removeMember check [ true true false ] access [ true false false false POST ]
func (a *Actions) RemoveMember(state state.IState, input inputs_points.RemoveMemberInput) (any, error) {
	trx := state.Trx()
	if trx.GetLink("admin::"+state.Info().PointId()+"::"+state.Info().UserId()) != "true" {
		return nil, errors.New("you are not admin")
	}
	if trx.GetLink("member::"+state.Info().PointId()+"::"+input.UserId) != "true" {
		return nil, errors.New("member not found")
	}
	trx.DelKey("link::member::" + state.Info().PointId() + "::" + input.UserId)
	trx.DelKey("link::memberof::" + input.UserId + "::" + state.Info().PointId())
	a.App.Tools().Signaler().LeaveGroup(state.Info().PointId(), input.UserId)
	user := model.User{Id: input.UserId}.Pull(trx)
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/removeMember", state.Info().PointId(), updates_points.AddMember{PointId: state.Info().PointId(), User: user}, true, []string{state.Info().UserId()})
	}, false)
	return outputs_points.AddMemberOutput{}, nil
}

// Create /points/create check [ true false false ] access [ true false false false POST ]
func (a *Actions) Create(state state.IState, input inputs_points.CreateInput) (any, error) {
	trx := state.Trx()
	orig := state.Source()
	if input.Origin() == "global" {
		orig = "global"
	}
	if input.ParentId != "" {
		if !trx.HasObj("Point", input.ParentId) {
			err := errors.New("parent point does not exist")
			log.Println(err)
			return nil, err
		}
		if trx.GetLink("admin::"+input.ParentId+"::"+state.Info().UserId()) != "true" {
			err := errors.New("access to point denied")
			log.Println(err)
			return nil, err
		}
	}
	point := model.Point{Id: a.App.Tools().Storage().GenId(trx, orig), Tag: input.Tag, IsPublic: *input.IsPublic, PersHist: *input.PersHist, ParentId: input.ParentId}
	point.Push(trx)
	trx.PutLink("memberof::"+state.Info().UserId()+"::"+point.Id, "true")
	trx.PutLink("member::"+point.Id+"::"+state.Info().UserId(), "true")
	trx.PutLink("admin::"+point.Id+"::"+state.Info().UserId(), "true")
	if input.Tag == "1-to-1" {
		if len(input.Members) > 2 {
			return nil, errors.New("1-to-1 chat can not have more than 2 members")
		} else if len(input.Members) == 2 && !input.Members[state.Info().UserId()] {
			return nil, errors.New("1-to-1 chat can not have more than 2 members")
		} else if len(input.Members) == 1 && input.Members[state.Info().UserId()] {
			return nil, errors.New("1-to-1 chat can not have more than 2 members")
		} else if len(input.Members) == 0 {
			return nil, errors.New("1-to-1 chat can not have more than 2 members")
		}
	}
	if input.Members != nil {
		for userId, isAdmin := range input.Members {
			trx.PutLink("memberof::"+userId+"::"+point.Id, "true")
			trx.PutLink("member::"+point.Id+"::"+userId, "true")
			if isAdmin {
				trx.PutLink("admin::"+point.Id+"::"+userId, "true")
			}
		}
	}
	if input.Metadata != nil {
		trx.PutJson("PointMeta::"+point.Id, "metadata", input.Metadata, false)
	}
	meta, err := trx.GetJson("PointMeta::"+point.Id, "metadata.public.profile")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if meta["title"] == nil {
		err := errors.New("title can't be empty")
		log.Println(err)
		return nil, err
	}
	if meta["avatar"] == nil {
		err := errors.New("avatar can't be empty")
		log.Println(err)
		return nil, err
	}
	trx.PutIndex("Point", "title", "id", point.Id+"->"+meta["title"].(string), []byte(point.Id))
	a.App.Tools().Signaler().JoinGroup(point.Id, state.Info().UserId())
	return outputs_points.CreateOutput{Point: point}, nil
}

// Update /points/update check [ true true false ] access [ true false false false PUT ]
func (a *Actions) Update(state state.IState, input inputs_points.UpdateInput) (any, error) {
	trx := state.Trx()
	if trx.GetLink("admin::"+state.Info().PointId()+"::"+state.Info().UserId()) != "true" {
		return nil, errors.New("you are not admin")
	}
	point := model.Point{Id: state.Info().PointId()}.Pull(trx)
	if input.IsPublic != nil {
		point.IsPublic = *input.IsPublic
	}
	if input.PersHist != nil {
		point.PersHist = *input.PersHist
	}
	if input.Metadata != nil {
		meta, err := trx.GetJson("PointMeta::"+point.Id, "metadata.public.profile")
		if err != nil {
			log.Println(err)
			return nil, err
		}
		trx.DelIndex("Point", "title", "id", point.Id+"->"+meta["title"].(string))
		trx.PutJson("PointMeta::"+point.Id, "metadata", input.Metadata, true)
		meta, err = trx.GetJson("PointMeta::"+point.Id, "metadata.public.profile")
		if err != nil {
			log.Println(err)
			return nil, err
		}
		trx.PutIndex("Point", "title", "id", point.Id+"->"+meta["title"].(string), []byte(point.Id))
	}
	point.Push(trx)
	meta, err := trx.GetJson("PointMeta::"+point.Id, "metadata.public.profile")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if meta["title"] == nil {
		err := errors.New("title can't be empty")
		log.Println(err)
		return nil, err
	}
	if meta["avatar"] == nil {
		err := errors.New("avatar can't be empty")
		log.Println(err)
		return nil, err
	}
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/update", point.Id, updates_points.Update{Point: point}, true, []string{state.Info().UserId()})
	}, false)
	return outputs_points.UpdateOutput{Point: point}, nil
}

// Delete /points/delete check [ true true false ] access [ true false false false DELETE ]
func (a *Actions) Delete(state state.IState, input inputs_points.DeleteInput) (any, error) {
	trx := state.Trx()
	if len(trx.GetColumn("Point", state.Info().PointId(), "|")) == 0 {
		return nil, errors.New("point does not exist")
	}
	if trx.GetLink("admin::"+state.Info().PointId()+"::"+state.Info().UserId()) != "true" {
		return nil, errors.New("you are not admin")
	}
	point := model.Point{Id: state.Info().PointId()}.Pull(trx)
	meta, err := trx.GetJson("PointMeta::"+point.Id, "metadata.public.profile")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	trx.DelIndex("Point", "title", "id", point.Id+"->"+meta["title"].(string))
	point.Delete(trx)
	members, _ := trx.GetLinksList("member::"+point.Id+"::", 0, 0)
	usersList := []string{}
	for _, member := range members {
		parts := strings.Split(member, "::")
		trx.DelKey("link::memberof::" + parts[1] + "::" + parts[2])
		trx.DelKey("link::" + member)
		usersList = append(usersList, parts[1])
	}
	admins, _ := trx.GetLinksList("admin::"+point.Id+"::", 0, 0)
	for _, admin := range admins {
		trx.DelKey("link::" + admin)
	}
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/delete", point.Id, updates_points.Delete{Point: point}, true, []string{state.Info().UserId()})
		for _, user := range usersList {
			a.App.Tools().Signaler().LeaveGroup(point.Id, user)
		}
	}, false)
	return outputs_points.DeleteOutput{Point: point}, nil
}

// Meta /points/meta check [ true false false ] access [ true false false false GET ]
func (a *Actions) Meta(state state.IState, input inputs_points.MetaInput) (any, error) {
	trx := state.Trx()
	if !trx.HasObj("Point", input.PointId) {
		return nil, errors.New("point not found")
	}
	isMember := a.App.Tools().Security().HasAccessToPoint(state.Info().PointId(), input.PointId)
	if !isMember && strings.HasPrefix(input.Path, "private.") {
		return nil, errors.New("access denied")
	}
	metadata, err := trx.GetJson("PointMeta::"+input.PointId, "metadata."+input.Path)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return map[string]any{"metadata": metadata}, nil
}

// Get /points/get check [ true false false ] access [ true false false false GET ]
func (a *Actions) Get(state state.IState, input inputs_points.GetInput) (any, error) {
	trx := state.Trx()
	if !trx.HasObj("Point", input.PointId) {
		return nil, errors.New("point not found")
	}
	point := model.Point{Id: input.PointId}.Pull(trx)
	if point.IsPublic {
		result := map[string]any{
			"id":       point.Id,
			"parentId": point.ParentId,
			"isPublic": point.IsPublic,
			"persHist": point.PersHist,
			"tag":      point.Tag,
		}
		if input.IncludeMeta {
			metadata, err := trx.GetJson("PointMeta::"+point.Id, "metadata")
			if err != nil {
				log.Println(err)
				return nil, err
			}
			result["metadata"] = metadata
		}
		return outputs_points.GetOutput{Point: result}, nil
	}
	if trx.GetLink("member::"+input.PointId+"::"+state.Info().UserId()) != "true" {
		return nil, errors.New("access to private point denied")
	}
	lastPacket := map[string]any{}
	lpData, err := trx.GetJson("PointMeta::"+point.Id, "metadata.public.lastPacket")
	if err == nil {
		lastPacket = lpData
	}
	result := map[string]any{
		"id":         point.Id,
		"parentId":   point.ParentId,
		"isPublic":   point.IsPublic,
		"persHist":   point.PersHist,
		"tag":        point.Tag,
		"lastPacket": lastPacket,
	}
	if input.IncludeMeta {
		metadata, err := trx.GetJson("PointMeta::"+point.Id, "metadata")
		if err != nil {
			log.Println(err)
			return nil, err
		}
		result["metadata"] = metadata
	}

	return outputs_points.GetOutput{Point: result}, nil
}

// Read /points/read check [ true false false ] access [ true false false false GET ]
func (a *Actions) Read(state state.IState, input inputs_points.ReadInput) (any, error) {
	trx := state.Trx()
	points, err := model.Point{}.List(trx, "memberof::"+state.Info().UserId()+"::", input.Offset, input.Count)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	results := []map[string]any{}
	for _, point := range points {
		lastPacket := map[string]any{}
		lpData, err := trx.GetJson("PointMeta::"+point.Id, "metadata.public.lastPacket")
		if err == nil {
			lastPacket = lpData
		}
		result := map[string]any{
			"id":         point.Id,
			"parentId":   point.ParentId,
			"isPublic":   point.IsPublic,
			"persHist":   point.PersHist,
			"tag":        point.Tag,
			"lastPacket": lastPacket,
		}
		meta, err := trx.GetJson("PointMeta::"+point.Id, "metadata.public.profile")
		if err != nil {
			log.Println(err)
			return nil, err
		}
		result["title"] = meta["title"]
		result["avatar"] = meta["avatar"]
		results = append(results, result)
	}
	return outputs_points.ReadOutput{Points: results}, nil
}

// Join /points/join check [ true false false ] access [ true false false false POST ]
func (a *Actions) Join(state state.IState, input inputs_points.JoinInput) (any, error) {
	trx := state.Trx()
	if !trx.HasObj("Point", input.PointId) {
		return nil, errors.New("point not found")
	}
	point := model.Point{Id: input.PointId}.Pull(trx)
	if !point.IsPublic {
		return nil, errors.New("point is private")
	}
	trx.PutLink("member::"+point.Id+"::"+state.Info().UserId(), "true")
	trx.PutLink("memberof::"+state.Info().UserId()+"::"+point.Id, "true")
	a.App.Tools().Signaler().JoinGroup(point.Id, state.Info().UserId())
	user := model.User{Id: state.Info().UserId()}.Pull(trx)
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/join", point.Id, updates_points.Join{PointId: point.Id, User: user}, true, []string{state.Info().UserId()})
	}, false)
	return outputs_points.JoinOutput{}, nil
}

// Signal /points/signal check [ true true true ] access [ true false false false POST ]
func (a *Actions) Signal(state state.IState, input inputs_points.SignalInput) (any, error) {
	trx := state.Trx()
	point := model.Point{Id: state.Info().PointId()}.Pull(trx)
	user := model.User{Id: state.Info().UserId()}.Pull(trx)
	t := time.Now().UnixMilli()
	if input.Type == "broadcast" {
		var p = updates_points.Send{Action: "broadcast", Point: point, User: user, Data: input.Data, Time: t}
		if point.PersHist {
			insertedId := a.App.Tools().Storage().LogTimeSieries(point.Id, user.Id, input.Data, t)
			trx.PutJson("PointMeta::"+point.Id, "metadata.public.lastPacket", map[string]any{
				"id":     insertedId,
				"data":   input.Data,
				"userId": state.Info().UserId(),
				"time":   t,
			}, false)
		}
		a.App.Tools().Signaler().SignalGroup("points/signal", point.Id, p, true, []string{state.Info().UserId()})
		return outputs_points.SignalOutput{Passed: true}, nil
	} else if input.Type == "single" {
		if trx.GetLink("member::"+point.Id+"::"+input.UserId) == "true" {
			var p = updates_points.Send{Action: "single", Point: point, User: user, Data: input.Data, Time: t}
			if point.PersHist {
				insertedId := a.App.Tools().Storage().LogTimeSieries(point.Id, user.Id, input.Data, t)
				trx.PutJson("PointMeta::"+point.Id, "metadata.public.lastPacket", map[string]any{
					"id":     insertedId,
					"data":   input.Data,
					"userId": state.Info().UserId(),
					"time":   t,
				}, false)
			}
			a.App.Tools().Signaler().SignalUser("points/signal", input.UserId, p, true)
			return outputs_points.SignalOutput{Passed: true}, nil
		}
	}
	return outputs_points.SignalOutput{Passed: false}, nil
}

// History /points/history check [ true true true ] access [ true false false false POST ]
func (a *Actions) History(state state.IState, input inputs_points.HistoryInput) (any, error) {
	return outputs_points.HistoryOutput{Packets: a.App.Tools().Storage().ReadPointLogs(state.Info().PointId(), input.BeforeId, input.Count)}, nil
}

// List /points/list check [ true false false ] access [ true false false false GET ]
func (a *Actions) List(state state.IState, input inputs_points.ListInput) (any, error) {
	trx := state.Trx()
	points, err := model.Point{}.Search(trx, input.Offset, input.Count, input.Query, map[string]string{"isPublic": string([]byte{0x01})})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	results := []map[string]any{}
	for _, point := range points {
		result := map[string]any{
			"id":       point.Id,
			"parentId": point.ParentId,
			"isPublic": point.IsPublic,
			"persHist": point.PersHist,
			"tag":      point.Tag,
		}
		meta, err := trx.GetJson("PointMeta::"+point.Id, "metadata.public.profile")
		if err != nil {
			log.Println(err)
			return nil, err
		}
		result["title"] = meta["title"]
		result["avatar"] = meta["avatar"]
		results = append(results, result)
	}
	return map[string]any{"points": results}, nil
}
