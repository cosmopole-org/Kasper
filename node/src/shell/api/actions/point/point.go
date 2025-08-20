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
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
)

type LockHolder struct {
	Lock sync.Mutex
}

type Actions struct {
	App           core.ICore
	Locks         cmap.ConcurrentMap[string, *LockHolder]
	OneToOneLocks cmap.ConcurrentMap[string, *LockHolder]
}

func Install(a *Actions) error {
	a.Locks = cmap.New[*LockHolder]()
	a.OneToOneLocks = cmap.New[*LockHolder]()
	return nil
}

// AddApp /points/addApp check [ true true false ] access [ true false false false POST ]
func (a *Actions) AddApp(state state.IState, input inputs_points.AddAppInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
	if !trx.HasObj("App", input.AppId) {
		return nil, errors.New("app not found")
	}
	app := model.App{Id: input.AppId}.Pull(trx)

	machines, err := model.User{}.List(trx, "appMachines::"+input.AppId+"::", map[string]string{})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	vms, err := model.Vm{}.List(trx, "appMachines::"+input.AppId+"::")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	vmMap := map[string]model.Vm{}
	for _, vm := range vms {
		vmMap[vm.MachineId] = vm
	}
	macMap := map[string]model.User{}
	for _, mac := range machines {
		macMap[mac.Id] = mac
	}
	m := map[string]*updates_points.Fn{}
	uniqueMacs := map[string][]string{}
	for _, machine := range input.MachinesMeta {
		mac := macMap[machine.MachineId]
		vm := vmMap[machine.MachineId]
		fn := &updates_points.Fn{
			UserId:     mac.Id,
			Typ:        mac.Typ,
			Username:   mac.Username,
			PublicKey:  mac.PublicKey,
			Name:       mac.Name,
			AppId:      vm.AppId,
			Runtime:    vm.Runtime,
			Path:       vm.Path,
			Comment:    vm.Comment,
			Metadata:   machine.Metadata,
			Identifier: machine.Identifier,
		}
		m[fn.UserId+"::"+fn.Identifier] = fn
		trx.PutJson("FnMeta::"+state.Info().PointId()+"::"+fn.AppId+"::"+fn.UserId+"::"+machine.Identifier, "metadata", machine.Metadata, true)
		trx.PutLink("pointAppMachine::"+state.Info().PointId()+"::"+app.Id+"::"+machine.MachineId+"::"+machine.Identifier, "true")
		uniqueMacs[fn.UserId] = append(uniqueMacs[fn.UserId], machine.Identifier)
	}
	for uniMacId, _ := range uniqueMacs {
		trx.PutLink("member::"+state.Info().PointId()+"::"+uniMacId, "true")
		trx.PutLink("memberof::"+uniMacId+"::"+state.Info().PointId(), "true")
		a.App.Tools().Signaler().JoinGroup(state.Info().PointId(), uniMacId)
	}
	trx.PutLink("pointApp::"+state.Info().PointId()+"::"+app.Id, "true")
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/addApp", state.Info().PointId(), updates_points.AddApp{PointId: state.Info().PointId(), App: app, Machines: m}, true, []string{})
	}, false)
	return outputs_points.AddMemberOutput{}, nil
}

// ListPointApps /points/listApps check [ true true false ] access [ true false false false POST ]
func (a *Actions) ListPointApps(state state.IState, input inputs_points.ListPointAppsInput) (any, error) {
	trx := state.Trx()
	prefix := "pointAppMachine::" + state.Info().PointId() + "::"
	machineLinks, err := trx.GetLinksList(prefix, 0, 1000)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	fns := map[string]*updates_points.Fn{}
	apps := map[string]model.App{}
	for _, mlink := range machineLinks {
		parts := strings.Split(mlink[len(prefix):], "::")
		appId := parts[0]
		machineId := parts[1]
		identifier := parts[2]
		machine := model.User{Id: machineId}.Pull(trx, true)
		metadata, err := trx.GetJson("FnMeta::"+state.Info().PointId()+"::"+appId+"::"+machineId+"::"+identifier, "metadata")
		if err != nil {
			log.Println(err)
		}
		meta, err := trx.GetJson("MachineMeta::"+machineId, "metadata")
		if err != nil {
			log.Println(err)
			meta = map[string]any{}
		}
		maps.Copy(metadata, meta)
		vm := model.Vm{MachineId: machineId}.Pull(trx)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		fn := &updates_points.Fn{
			UserId:     machine.Id,
			Typ:        machine.Typ,
			Username:   machine.Username,
			PublicKey:  machine.PublicKey,
			Name:       machine.Name,
			AppId:      vm.AppId,
			Runtime:    vm.Runtime,
			Path:       vm.Path,
			Comment:    vm.Comment,
			Metadata:   metadata,
			Identifier: identifier,
		}

		if _, ok := apps[fn.AppId]; !ok {
			apps[fn.AppId] = model.App{Id: fn.AppId}.Pull(trx, true)
		}
		fns[fn.UserId+"::"+fn.Identifier] = fn
	}
	return outputs_points.ListPointAppsOutput{Apps: apps, Machines: fns}, nil
}

// UpdateMachine /points/updateMachine check [ true true false ] access [ true false false false POST ]
func (a *Actions) UpdateMachine(state state.IState, input inputs_points.UpdateMachineInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
	if !trx.HasObj("App", input.AppId) {
		return nil, errors.New("app not found")
	}
	app := model.App{Id: input.AppId}.Pull(trx)
	trx.PutJson("FnMeta::"+state.Info().PointId()+"::"+input.AppId+"::"+input.MachineMeta.MachineId+"::"+input.MachineMeta.Identifier, "metadata", input.MachineMeta.Metadata, false)
	machine := model.User{Id: input.MachineMeta.MachineId}.Pull(trx)
	vm := model.Vm{MachineId: input.MachineMeta.MachineId}.Pull(trx)
	fn := updates_points.Fn{
		UserId:     machine.Id,
		Typ:        machine.Typ,
		Username:   machine.Username,
		PublicKey:  machine.PublicKey,
		Name:       machine.Name,
		AppId:      vm.AppId,
		Runtime:    vm.Runtime,
		Path:       vm.Path,
		Comment:    vm.Comment,
		Metadata:   input.MachineMeta.Metadata,
		Identifier: input.MachineMeta.Identifier,
	}
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/updateMachine", state.Info().PointId(), updates_points.UpdateApp{PointId: state.Info().PointId(), App: app, Machine: fn}, true, []string{state.Info().UserId()})
	}, false)
	return outputs_points.AddMemberOutput{}, nil
}

// RemoveApp /points/removeApp check [ true true false ] access [ true false false false POST ]
func (a *Actions) RemoveApp(state state.IState, input inputs_points.RemoveAppInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
	if !trx.HasObj("App", input.AppId) {
		return nil, errors.New("app not found")
	}
	app := model.App{Id: input.AppId}.Pull(trx)
	machines, err := model.User{}.List(trx, "appMachines::"+input.AppId+"::", map[string]string{})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	vms, err := model.Vm{}.List(trx, "appMachines::"+input.AppId+"::")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	vmMap := map[string]model.Vm{}
	for _, vm := range vms {
		vmMap[vm.MachineId] = vm
	}
	macArr := strings.Split(trx.GetLink("pointAppMachines::"+state.Info().PointId()+"::"+app.Id), ",")
	for _, machine := range machines {
		if slices.Contains(macArr, machine.Id) {
			trx.DelKey("link::member::" + state.Info().PointId() + "::" + machine.Id)
			trx.DelKey("link::memberof::" + machine.Id + "::" + state.Info().PointId())
			a.App.Tools().Signaler().LeaveGroup(state.Info().PointId(), machine.Id)
		}
	}
	prefix := "pointAppMachine::" + state.Info().PointId() + "::" + app.Id + "::"
	arr, err := trx.GetLinksList(prefix, 0, 1000)
	if err != nil {
		log.Println(err)
	} else {
		for _, key := range arr {
			parts := strings.Split(key[len(prefix):], "::")
			trx.DelJson("FnMeta::"+state.Info().PointId()+"::"+input.AppId+"::"+parts[0]+"::"+parts[1], "metadata")
			trx.DelKey(key)
		}
	}
	trx.DelKey("link::pointApp::" + state.Info().PointId() + "::" + app.Id)
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/removeApp", state.Info().PointId(), updates_points.AddApp{PointId: state.Info().PointId(), App: app}, true, []string{state.Info().UserId()})
	}, false)
	return outputs_points.AddMemberOutput{}, nil
}

// AddMachine /points/addMachine check [ true true false ] access [ true false false false POST ]
func (a *Actions) AddMachine(state state.IState, input inputs_points.AddMachineInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
	if !trx.HasObj("App", input.AppId) {
		return nil, errors.New("app not found")
	}
	app := model.App{Id: input.AppId}.Pull(trx)
	machine := model.User{Id: input.MachineMeta.MachineId}.Pull(trx)
	vm := model.Vm{MachineId: input.MachineMeta.MachineId}.Pull(trx)
	meta, err := trx.GetJson("MachineMeta::"+vm.MachineId, "metadata")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	maps.Copy(input.MachineMeta.Metadata, meta)
	fn := updates_points.Fn{
		UserId:     machine.Id,
		Typ:        machine.Typ,
		Username:   machine.Username,
		PublicKey:  machine.PublicKey,
		Name:       machine.Name,
		AppId:      vm.AppId,
		Runtime:    vm.Runtime,
		Path:       vm.Path,
		Comment:    vm.Comment,
		Identifier: input.MachineMeta.Identifier,
		Metadata:   input.MachineMeta.Metadata,
	}
	trx.PutJson("FnMeta::"+state.Info().PointId()+"::"+fn.AppId+"::"+fn.UserId+"::"+input.MachineMeta.Identifier, "metadata", input.MachineMeta.Metadata, true)
	trx.PutLink("member::"+state.Info().PointId()+"::"+input.MachineMeta.MachineId, "true")
	trx.PutLink("memberof::"+input.MachineMeta.MachineId+"::"+state.Info().PointId(), "true")
	trx.PutLink("pointAppMachine::"+state.Info().PointId()+"::"+input.AppId+"::"+input.MachineMeta.MachineId+"::"+input.MachineMeta.Identifier, "true")
	a.App.Tools().Signaler().JoinGroup(state.Info().PointId(), input.MachineMeta.MachineId)
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/addMachine", state.Info().PointId(), updates_points.AddMachine{PointId: state.Info().PointId(), App: app, Machine: fn}, true, []string{})
	}, false)
	return outputs_points.AddMemberOutput{}, nil
}

// RemoveMachine /points/removeMachine check [ true true false ] access [ true false false false POST ]
func (a *Actions) RemoveMachine(state state.IState, input inputs_points.RemoveMachineInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
	if !trx.HasObj("App", input.AppId) {
		return nil, errors.New("app not found")
	}
	if trx.GetLink("pointAppMachine::"+state.Info().PointId()+"::"+input.AppId+"::"+input.MachineId+"::"+input.Identifier) == "" {
		return nil, errors.New("machine with this identifier does not exist in point")
	}
	app := model.App{Id: input.AppId}.Pull(trx)
	machine := model.User{Id: input.MachineId}.Pull(trx)
	vm := model.Vm{MachineId: input.MachineId}.Pull(trx)
	fn := updates_points.Fn{
		UserId:     machine.Id,
		Typ:        machine.Typ,
		Username:   machine.Username,
		PublicKey:  machine.PublicKey,
		Name:       machine.Name,
		AppId:      vm.AppId,
		Runtime:    vm.Runtime,
		Path:       vm.Path,
		Comment:    vm.Comment,
		Identifier: input.Identifier,
	}
	trx.DelJson("FnMeta::"+state.Info().PointId()+"::"+fn.AppId+"::"+fn.UserId+"::"+input.Identifier, "metadata")
	trx.DelKey("link::pointAppMachine::" + state.Info().PointId() + "::" + input.AppId + "::" + input.MachineId + "::" + input.Identifier)
	if arr, err := trx.GetLinksList("pointAppMachine::"+state.Info().PointId()+"::"+input.AppId+"::"+input.MachineId+"::", 0, 100); err == nil && len(arr) == 0 {
		trx.DelKey("link::member::" + state.Info().PointId() + "::" + input.MachineId)
		trx.DelKey("link::memberof::" + input.MachineId + "::" + state.Info().PointId())
		a.App.Tools().Signaler().LeaveGroup(state.Info().PointId(), input.MachineId)
	}
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/removeMachine", state.Info().PointId(), updates_points.AddMachine{PointId: state.Info().PointId(), App: app, Machine: fn}, true, []string{})
	}, false)
	return outputs_points.AddMemberOutput{}, nil
}

// AddMember /points/addMember check [ true true false ] access [ true false false false POST ]
func (a *Actions) AddMember(state state.IState, input inputs_points.AddMemberInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
	if !trx.HasObj("User", input.UserId) {
		return nil, errors.New("user not found")
	}
	point := model.Point{Id: state.Info().PointId()}.Pull(trx)
	if point.Tag == "home" {
		return nil, errors.New("home is not extendable")
	}
	if trx.GetLink("member::"+state.Info().PointId()+"::"+input.UserId) == "true" {
		return nil, errors.New("membership already exists")
	}
	trx.PutLink("member::"+state.Info().PointId()+"::"+input.UserId, "true")
	trx.PutLink("memberof::"+input.UserId+"::"+state.Info().PointId(), "true")
	point.MemberCount++
	point.Push(trx)
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
	point := model.Point{Id: state.Info().PointId()}.Pull(trx)
	if point.Tag == "home" {
		return nil, errors.New("home is not extendable")
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
	members, err := model.User{}.List(trx, "member::"+state.Info().PointId()+"::", map[string]string{"type": "human"})
	if err != nil {
		return nil, err
	}
	membersArr := []map[string]any{}
	for _, member := range members {
		if member.Typ == "human" {
			metadata, err := trx.GetJson("UserMeta::"+member.Id, "metadata.public.profile")
			if err != nil {
				log.Println(err)
				return nil, err
			}
			membersArr = append(membersArr, map[string]any{
				"id":        member.Id,
				"publicKey": member.PublicKey,
				"type":      member.Typ,
				"username":  member.Username,
				"name":      metadata["name"],
			})
		}
	}
	return outputs_points.ReadMemberOutput{Members: membersArr}, nil
}

// RemoveMember /points/removeMember check [ true true false ] access [ true false false false POST ]
func (a *Actions) RemoveMember(state state.IState, input inputs_points.RemoveMemberInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
	if trx.GetLink("admin::"+state.Info().PointId()+"::"+state.Info().UserId()) != "true" {
		return nil, errors.New("you are not admin")
	}
	if trx.GetLink("member::"+state.Info().PointId()+"::"+input.UserId) != "true" {
		return nil, errors.New("member not found")
	}
	point := model.Point{Id: state.Info().PointId()}.Pull(trx)
	if point.Tag == "home" {
		return nil, errors.New("home is not extendable")
	}
	if trx.GetLink("member::"+state.Info().PointId()+"::"+input.UserId) != "true" {
		return nil, errors.New("membership does exist")
	}
	trx.DelKey("link::member::" + state.Info().PointId() + "::" + input.UserId)
	trx.DelKey("link::memberof::" + input.UserId + "::" + state.Info().PointId())
	point.MemberCount--
	point.Push(trx)
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
	input.Members[state.Info().UserId()] = true
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
		for k := range input.Members {
			input.Members[k] = true
		}
		ids := []string{}
		for k := range input.Members {
			ids = append(ids, k)
		}
		slices.Sort(ids)
		a.OneToOneLocks.SetIfAbsent(ids[0]+"<->"+ids[1], &LockHolder{})
		locker, _ := a.OneToOneLocks.Get(ids[0] + "<->" + ids[1])
		locker.Lock.Lock()
		defer locker.Lock.Unlock()
		if pointId := trx.GetLink("1-to-1-map::" + ids[0] + "<->" + ids[1]); pointId != "" {
			point := model.Point{Id: pointId}.Pull(trx, true)
			return outputs_points.CreateOutput{Point: outputs_points.AdminPoiint{Point: point, Admin: true}}, nil
		}
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
	point := model.Point{Id: a.App.Tools().Storage().GenId(trx, orig), MemberCount: int32(len(input.Members)), Tag: input.Tag, IsPublic: *input.IsPublic, PersHist: *input.PersHist, ParentId: input.ParentId}
	point.Push(trx)
	trx.PutLink("memberof::"+state.Info().UserId()+"::"+point.Id, "true")
	trx.PutLink("member::"+point.Id+"::"+state.Info().UserId(), "true")
	trx.PutLink("admin::"+point.Id+"::"+state.Info().UserId(), "true")
	trx.PutLink("adminof::"+state.Info().UserId()+"::"+point.Id, "true")
	if input.Members != nil {
		for userId, isAdmin := range input.Members {
			trx.PutLink("memberof::"+userId+"::"+point.Id, "true")
			trx.PutLink("member::"+point.Id+"::"+userId, "true")
			if isAdmin {
				trx.PutLink("admin::"+point.Id+"::"+userId, "true")
				trx.PutLink("adminof::"+userId+"::"+point.Id, "true")
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
	if input.Tag == "1-to-1" {
		ids := []string{}
		for k := range input.Members {
			ids = append(ids, k)
		}
		slices.Sort(ids)
		trx.PutLink("1-to-1-map::"+ids[0]+"<->"+ids[1], point.Id)
	}
	for memberId := range input.Members {
		a.App.Tools().Signaler().JoinGroup(point.Id, memberId)
	}
	point = point.Pull(trx, true)
	a.App.Tools().Signaler().SignalGroup("points/create", point.Id, updates_points.Delete{Point: point}, true, []string{state.Info().UserId()})
	return outputs_points.CreateOutput{Point: outputs_points.AdminPoiint{Point: point, Admin: true}}, nil
}

// Update /points/update check [ true true false ] access [ true false false false PUT ]
func (a *Actions) Update(state state.IState, input inputs_points.UpdateInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
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
	return outputs_points.UpdateOutput{Point: outputs_points.AdminPoiint{Point: point, Admin: true}}, nil
}

// Delete /points/delete check [ true true false ] access [ true false false false DELETE ]
func (a *Actions) Delete(state state.IState, input inputs_points.DeleteInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
	if len(trx.GetColumn("Point", state.Info().PointId(), "|")) == 0 {
		return nil, errors.New("point does not exist")
	}
	if trx.GetLink("admin::"+state.Info().PointId()+"::"+state.Info().UserId()) != "true" {
		return nil, errors.New("you are not admin")
	}
	point := model.Point{Id: state.Info().PointId()}.Pull(trx)
	if point.Tag == "home" {
		return nil, errors.New("your home can not be deleted")
	}
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
	prefix := "admin::" + point.Id + "::"
	admins, _ := trx.GetLinksList(prefix, 0, 0)
	for _, admin := range admins {
		trx.DelKey("link::" + admin)
		trx.DelKey("link::adminof::" + admin[len(prefix):] + "::" + point.Id)
	}
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/delete", point.Id, updates_points.Delete{Point: point}, true, []string{state.Info().UserId()})
		for _, user := range usersList {
			a.App.Tools().Signaler().LeaveGroup(point.Id, user)
		}
	}, false)
	a.Locks.Remove(state.Info().PointId())
	return outputs_points.DeleteOutput{Point: outputs_points.AdminPoiint{Point: point, Admin: true}}, nil
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
			"id":          point.Id,
			"parentId":    point.ParentId,
			"isPublic":    point.IsPublic,
			"persHist":    point.PersHist,
			"memberCount": point.MemberCount,
			"signalCount": point.SignalCount,
			"tag":         point.Tag,
		}
		if input.IncludeMeta {
			metadata, err := trx.GetJson("PointMeta::"+point.Id, "metadata")
			if err != nil {
				log.Println(err)
				return nil, err
			}
			result["metadata"] = metadata
		}
		if trx.GetLink("adminof::"+state.Info().UserId()+"::"+point.Id) == "true" {
			result["admin"] = true
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
		"id":          point.Id,
		"parentId":    point.ParentId,
		"isPublic":    point.IsPublic,
		"persHist":    point.PersHist,
		"memberCount": point.MemberCount,
		"signalCount": point.SignalCount,
		"tag":         point.Tag,
		"lastPacket":  lastPacket,
	}
	if input.IncludeMeta {
		metadata, err := trx.GetJson("PointMeta::"+point.Id, "metadata")
		if err != nil {
			log.Println(err)
			return nil, err
		}
		result["metadata"] = metadata
	}
	if trx.GetLink("adminof::"+state.Info().UserId()+"::"+point.Id) == "true" {
		result["admin"] = true
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
			"id":          point.Id,
			"parentId":    point.ParentId,
			"isPublic":    point.IsPublic,
			"persHist":    point.PersHist,
			"memberCount": point.MemberCount,
			"signalCount": point.SignalCount,
			"tag":         point.Tag,
			"lastPacket":  lastPacket,
		}
		meta, err := trx.GetJson("PointMeta::"+point.Id, "metadata.public.profile")
		if err != nil {
			log.Println(err)
			return nil, err
		}
		result["title"] = meta["title"]
		result["avatar"] = meta["avatar"]
		if trx.GetLink("adminof::"+state.Info().UserId()+"::"+point.Id) == "true" {
			result["admin"] = true
		}
		results = append(results, result)
	}
	return outputs_points.ReadOutput{Points: results}, nil
}

// Join /points/join check [ true false false ] access [ true false false false POST ]
func (a *Actions) Join(state state.IState, input inputs_points.JoinInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
	if !trx.HasObj("Point", input.PointId) {
		return nil, errors.New("point not found")
	}
	point := model.Point{Id: input.PointId}.Pull(trx)
	if !point.IsPublic {
		return nil, errors.New("point is private")
	}
	if trx.GetLink("member::"+point.Id+"::"+state.Info().UserId()) == "true" {
		return nil, errors.New("membership already eixsts")
	}
	trx.PutLink("member::"+point.Id+"::"+state.Info().UserId(), "true")
	trx.PutLink("memberof::"+state.Info().UserId()+"::"+point.Id, "true")
	point.MemberCount++
	point.Push(trx)
	a.App.Tools().Signaler().JoinGroup(point.Id, state.Info().UserId())
	user := model.User{Id: state.Info().UserId()}.Pull(trx)
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/join", point.Id, updates_points.Join{PointId: point.Id, User: user}, true, []string{state.Info().UserId()})
	}, false)
	return outputs_points.JoinOutput{}, nil
}

// Leave /points/leave check [ true true false ] access [ true false false false POST ]
func (a *Actions) Leave(state state.IState, input inputs_points.JoinInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
	if !trx.HasObj("Point", input.PointId) {
		return nil, errors.New("point not found")
	}
	point := model.Point{Id: input.PointId}.Pull(trx)
	if !point.IsPublic {
		return nil, errors.New("point is private")
	}
	if trx.GetLink("member::"+point.Id+"::"+state.Info().UserId()) != "true" {
		return nil, errors.New("membership doesn't eixst")
	}
	trx.DelKey("link::member::" + point.Id + "::" + state.Info().UserId())
	trx.DelKey("link::memberof::" + state.Info().UserId() + "::" + point.Id)
	if trx.GetLink("admin::"+point.Id+"::"+state.Info().UserId()) == "true" {
		trx.DelKey("link::admin::" + point.Id + "::" + state.Info().UserId())
		trx.DelKey("lnik::adminof::" + state.Info().UserId() + "::" + point.Id)
	}
	point.MemberCount--
	point.Push(trx)
	a.App.Tools().Signaler().LeaveGroup(point.Id, state.Info().UserId())
	user := model.User{Id: state.Info().UserId()}.Pull(trx)
	future.Async(func() {
		a.App.Tools().Signaler().SignalGroup("points/join", point.Id, updates_points.Join{PointId: point.Id, User: user}, true, []string{state.Info().UserId()})
	}, false)
	if point.MemberCount == 0 {
		meta, err := trx.GetJson("PointMeta::"+point.Id, "metadata.public.profile")
		if err != nil {
			log.Println(err)
			return nil, err
		}
		trx.DelIndex("Point", "title", "id", point.Id+"->"+meta["title"].(string))
		point.Delete(trx)
		a.Locks.Remove(state.Info().PointId())
	}
	return outputs_points.JoinOutput{}, nil
}

// Signal /points/signal check [ true true true ] access [ true false false false POST ]
func (a *Actions) Signal(state state.IState, input inputs_points.SignalInput) (any, error) {
	trx := state.Trx()
	a.Locks.SetIfAbsent(state.Info().PointId(), &LockHolder{})
	locker, _ := a.Locks.Get(state.Info().PointId())
	locker.Lock.Lock()
	defer locker.Lock.Unlock()
	point := model.Point{Id: state.Info().PointId()}.Pull(trx, true)
	user := model.User{Id: state.Info().UserId()}.Pull(trx, true)
	t := time.Now().UnixMilli()
	if input.Type == "broadcast" {
		if point.PersHist && !input.Temp {
			packet := a.App.Tools().Storage().LogTimeSieries(point.Id, user.Id, input.Data, t)
			trx.PutJson("PointMeta::"+point.Id, "metadata.public.lastPacket", packet, false)
			var p = updates_points.Send{Id: packet.Id, Action: "broadcast", Point: point, User: user, Data: input.Data, Time: t}
			point.SignalCount++
			point.Push(trx)
			future.Async(func() {
				a.App.Tools().Signaler().SignalGroup("points/signal", point.Id, p, true, []string{state.Info().UserId()})
			}, false)
			return outputs_points.SignalOutput{Passed: true, Packet: packet}, nil
		} else {
			var p = updates_points.Send{Action: "broadcast", Point: point, User: user, Data: input.Data, Time: t, IsTemp: true}
			future.Async(func() {
				a.App.Tools().Signaler().SignalGroup("points/signal", point.Id, p, true, []string{state.Info().UserId()})
			}, false)
			return outputs_points.SignalOutput{Passed: true}, nil
		}
	} else if input.Type == "single" {
		if trx.GetLink("member::"+point.Id+"::"+input.UserId) == "true" {
			if point.PersHist && !input.Temp {
				packet := a.App.Tools().Storage().LogTimeSieries(point.Id, user.Id, input.Data, t)
				trx.PutJson("PointMeta::"+point.Id, "metadata.public.lastPacket", packet, false)
				var p = updates_points.Send{Id: packet.Id, Action: "single", Point: point, User: user, Data: input.Data, Time: t}
				point.SignalCount++
				point.Push(trx)
				future.Async(func() {
					a.App.Tools().Signaler().SignalUser("points/signal", input.UserId, p, true)
				}, false)
				return outputs_points.SignalOutput{Passed: true, Packet: packet}, nil
			} else {
				var p = updates_points.Send{Action: "single", Point: point, User: user, Data: input.Data, Time: t, IsTemp: true}
				future.Async(func() {
					a.App.Tools().Signaler().SignalUser("points/signal", input.UserId, p, true)
				}, false)
				return outputs_points.SignalOutput{Passed: true}, nil
			}
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
			"id":          point.Id,
			"parentId":    point.ParentId,
			"isPublic":    point.IsPublic,
			"persHist":    point.PersHist,
			"memberCount": point.MemberCount,
			"tag":         point.Tag,
		}
		meta, err := trx.GetJson("PointMeta::"+point.Id, "metadata.public.profile")
		if err != nil {
			log.Println(err)
			return nil, err
		}
		result["title"] = meta["title"]
		result["avatar"] = meta["avatar"]
		if trx.GetLink("adminof::"+state.Info().UserId()+"::"+point.Id) == "true" {
			result["admin"] = true
		}
		results = append(results, result)
	}
	return map[string]any{"points": results}, nil
}
