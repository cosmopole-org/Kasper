package l3

import (
	"encoding/json"
	"errors"
	"kasper/src/abstract/models/action"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/input"
	"kasper/src/abstract/models/update"
	"kasper/src/core/module/actor/model/l2"
	modulelogger "kasper/src/core/module/logger"
	modulemodel "kasper/src/shell/layer1/model"
)

type Parse func(interface{}) (input.IInput, error)

type SecureAction struct {
	action.IAction
	core    core.ICore
	Guard   *Guard
	logger  *modulelogger.Logger
	Parsers map[string]Parse
}

func NewSecureAction(action action.IAction, guard *Guard, core core.ICore, logger *modulelogger.Logger, parsers map[string]Parse) *SecureAction {
	return &SecureAction{action, core, guard, logger, parsers}
}

func (a *SecureAction) HasGlobalParser() bool {
	return a.Parsers["*"] != nil
}

func (a *SecureAction) ParseInput(protocol string, raw interface{}) (input.IInput, error) {
	return a.Parsers[protocol](raw)
}

func (a *SecureAction) SecurlyActChain(userId string, packetId string, packetBinary []byte, packetSignature string, input input.IInput, origin string) {
	success, info := a.Guard.CheckValidity(a.core, packetBinary, packetSignature, userId, input.GetPointId())
	if !success {
		a.core.ExecBaseResponseOnChain(packetId, core.EmptyPayload{}, 403, "authorization failed", []update.Update{})
	} else {
		a.core.ModifyStateSecurly(false, info, func(s l2.State) {
			updates := []update.Update{}
			sc, res, err := a.Act(s, input)
			updates = s.Trx().Updates()
			if err != nil {
				a.core.ExecBaseResponseOnChain(packetId, core.EmptyPayload{}, 500, err.Error(), []update.Update{})
			} else {
				a.core.ExecBaseResponseOnChain(packetId, res, sc, "", updates)
			}
		})
	}
}

func (a *SecureAction) SecurelyAct(userId string, packetId string, packetBinary []byte, packetSignature string, input input.IInput, dummy string) (int, any, error) {
	origin := input.Origin()
	if origin == "" {
		origin = a.core.Id()
	}
	if origin == "global" {
		c := make(chan int, 1)
		var res any
		var sc int
		var e error
		a.core.ExecBaseRequestOnChain(a.Key(), userId, packetBinary, packetSignature, func(data []byte, resCode int, err error) {
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
		success, info := a.Guard.CheckValidity(a.core, packetBinary, packetSignature, userId, input.GetPointId())
		if !success {
			return -1, nil, errors.New("authorization failed")
		} else {
			var sc int
			var res any
			var err error
			a.core.ModifyStateSecurly(false, info, func(s l2.State) {
				sc, res, err = a.Act(s, input)
			})
			return sc, res, err
		}
	}
	success := a.Guard.CheckIdentity(a.core, packetBinary, packetSignature, userId)
	if !success {
		return -1, nil, errors.New("authorization failed")
	}
	cFed := make(chan int, 1)
	var scFed int
	var resFed any
	var errFed error
	if a.Key() == "/storage/download" {
		a.core.Tools().Network.Federation().SendInFederationFileReqByCallback(origin, modulemodel.OriginPacket{IsResponse: false, Key: a.Key(), UserId: userId, PointId: input.GetPointId(), Binary: packetBinary, Signature: packetSignature, RequestId: packetId}, func(path string, err error) {
			if err != nil {
				scFed = 0
				resFed = map[string]any{}
				errFed = err
			} else {
				scFed = 1
				resFed = modulemodel.Command{Value: "sendFile", Data: path}
				errFed = nil
			}
			cFed <- 1
		})
	} else {
		a.core.Tools().Network.Federation().SendInFederationPacketByCallback(origin, modulemodel.OriginPacket{IsResponse: false, Key: a.Key(), UserId: userId, PointId: input.GetPointId(), Binary: packetBinary, Signature: packetSignature, RequestId: packetId}, func(data []byte, resCode int, err error) {
			result := map[string]any{}
			json.Unmarshal(data, &result)
			scFed = resCode
			resFed = result
			errFed = err
			cFed <- 1
		})
	}
	<-cFed
	return scFed, resFed, errFed
}

func (a *SecureAction) SecurelyActFed(userId string, packetBinary []byte, packetSignature string, input input.IInput) (int, any, error) {
	success, info := a.Guard.CheckValidity(a.core, packetBinary, packetSignature, userId, input.GetPointId())
	if !success {
		return -1, nil, nil
	}
	var sc int
	var res any
	var err error
	a.core.ModifyStateSecurly(false, info, func(s l2.State) {
		sc, res, err = a.Act(s, input)
		if res != nil {
			executable, ok := res.(func() (any, error))
			if ok {
				o, e := executable()
				res = o
				err = e
			}
		}
	})
	return sc, res, err
}
