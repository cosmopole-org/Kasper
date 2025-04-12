package actor

import (
	"encoding/json"
	"errors"
	"kasper/src/abstract"
	modulelogger "kasper/src/core/module/logger"
	"kasper/src/shell/layer1/adapters"
	modulemodel "kasper/src/shell/layer1/model"
	statemodule "kasper/src/shell/layer1/module/state"
	toolbox2 "kasper/src/shell/layer1/module/toolbox"
)

type Parse func(interface{}) (abstract.IInput, error)

type SecureAction struct {
	abstract.IAction
	core    abstract.ICore
	Guard   *Guard
	logger  *modulelogger.Logger
	Parsers map[string]Parse
}

func NewSecureAction(action abstract.IAction, guard *Guard, core abstract.ICore, logger *modulelogger.Logger, parsers map[string]Parse) *SecureAction {
	return &SecureAction{action, core, guard, logger, parsers}
}

func (a *SecureAction) HasGlobalParser() bool {
	return a.Parsers["*"] != nil
}

func (a *SecureAction) ParseInput(protocol string, raw interface{}) (abstract.IInput, error) {
	return a.Parsers[protocol](raw)
}

func (a *SecureAction) SecurlyActChain(layer abstract.ILayer, token string, packetId string, input abstract.IInput, origin string) {
	success, info := a.Guard.ValidateByToken(layer, token, input.GetSpaceId(), input.GetTopicId(), input.GetMemberId())
	if !success {
		a.core.ExecBaseResponseOnChain(packetId, abstract.EmptyPayload{}, 403, "authorization failed", []abstract.Update{}, []abstract.CacheUpdate{})
	}
	s := layer.Sb().NewState(info, nil, origin).(statemodule.IStateL1)
	tb := abstract.UseToolbox[toolbox2.IToolboxL1](a.core.Get(2).Tools())
	var sc int
	var res any
	updates := []abstract.Update{}
	cacheUpdates := []abstract.CacheUpdate{}
	err := tb.Storage().DoTrx(func(i adapters.ITrx) error {
		s.SetTrx(i)
		statusCode, data, err := a.Act(s, input)
		sc = statusCode
		res = data
		updates = i.Updates()
		cacheUpdates = i.Mem().Updates()
		return err
	})
	if err != nil {
		a.core.ExecBaseResponseOnChain(packetId, abstract.EmptyPayload{}, 500, err.Error(), []abstract.Update{}, []abstract.CacheUpdate{})
	} else {
		a.core.ExecBaseResponseOnChain(packetId, res, sc, "", updates, cacheUpdates)
	}
}

func (a *SecureAction) SecurelyAct(layer abstract.ILayer, token string, packetId string, input abstract.IInput, dummy string) (int, any, error) {
	origin := input.Origin()
	if origin == "" {
		origin = a.core.Id()
	}
	if origin == "global" {
		c := make(chan int, 1)
		var res any
		var sc int
		var e error
		a.core.ExecBaseRequestOnChain(a.Key(), input, layer.Index()+1, token, func(data []byte, resCode int, err error) {
			result := map[string]any{}
			json.Unmarshal(data, &result)
			res = result
			sc = resCode
			e = err
			c <- 1
		})
		<-c
		return sc, res, e
	}
	if a.core.Id() == origin {
		success, info := a.Guard.ValidateByToken(layer, token, input.GetSpaceId(), input.GetTopicId(), input.GetMemberId())
		if !success {
			return -1, nil, errors.New("authorization failed")
		}
		s := layer.Sb().NewState(info, nil, dummy).(statemodule.IStateL1)
		tb := abstract.UseToolbox[toolbox2.IToolboxL1](a.core.Get(2).Tools())
		var sc int
		var res any
		var err error
		tb.Storage().DoTrx(func(i adapters.ITrx) error {
			s.SetTrx(i)
			sc, res, err = a.Act(s, input)
			return err
		})
		if res != nil {
			executable, ok := res.(func() (any, error))
			if ok {
				o, e := executable()
				return sc, o, e
			}
		}
		return sc, res, err
	}
	success, userId := a.Guard.ValidateOnlyToken(layer, token)
	if !success {
		return -1, nil, nil
	}
	toolbox := abstract.UseToolbox[*toolbox2.ToolboxL1](a.core.Get(1).Tools())
	data, err := json.Marshal(input)
	if err != nil {
		a.logger.Println(err)
	}
	cFed := make(chan int, 1)
	var scFed int
	var resFed any
	var errFed error
	toolbox.Federation().SendInFederationByCallback(origin, modulemodel.OriginPacket{Layer: layer.Index() + 1, IsResponse: false, Key: a.Key(), UserId: userId, SpaceId: input.GetSpaceId(), TopicId: input.GetTopicId(), Data: string(data), RequestId: packetId}, func(data []byte, resCode int, err error) {
		result := map[string]any{}
		json.Unmarshal(data, &result)
		scFed = resCode
		resFed = result
		errFed = err
		cFed <- 1
	})
	<-cFed
	return scFed, resFed, errFed
}

func (a *SecureAction) SecurelyActFed(layer abstract.ILayer, userId string, input abstract.IInput) (int, any, error) {
	success, info := a.Guard.ValidateByUserId(userId, input.GetSpaceId(), input.GetTopicId(), input.GetMemberId())
	if !success {
		return -1, nil, nil
	}
	s := layer.Sb().NewState(info, nil, "").(statemodule.IStateL1)
	tb := abstract.UseToolbox[toolbox2.IToolboxL1](a.core.Get(2).Tools())
	var sc int
	var res any
	var err error
	tb.Storage().DoTrx(func(i adapters.ITrx) error {
		s.SetTrx(i)
		sc, res, err = a.Act(s, input)
		return err
	})
	if res != nil {
		executable, ok := res.(func() (any, error))
		if ok {
			o, e := executable()
			return sc, o, e
		}
	}
	return sc, res, err
}
