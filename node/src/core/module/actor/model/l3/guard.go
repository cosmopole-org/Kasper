package l3

import (
	"kasper/src/abstract/models/core"
	model "kasper/src/core/module/actor/model/l1"
)

type Guard struct {
	IsUser    bool `json:"isUser"`
	IsInSpace bool `json:"isInSpace"`
	IsInTopic bool `json:"isInTopic"`
}

func (g *Guard) CheckValidity(app core.ICore, packet []byte, signature string, userId string, pointId string) (bool, *model.Info) {
	if !g.IsUser {
		return true, model.NewInfo("", "")
	}
	identified, _, isGod := app.Tools().Security.AuthWithSignature(userId, packet, signature)
	if !identified {
		return false, &model.Info{}
	}
	if !g.IsInSpace {
		return true, model.NewGodInfo(userId, "", isGod)
	}
    hasAccess := app.Tools().Security.HasAccessToPoint(userId, pointId)
	if hasAccess {
		return false, &model.Info{}
	}
	return true, model.NewGodInfo(userId, pointId, isGod)
}

func (g *Guard) CheckIdentity(app core.ICore, packet []byte, signature string, userId string) bool {
	if !g.IsUser {
		return true
	}
	identified, _, _ := app.Tools().Security.AuthWithSignature(userId, packet, signature)
	if !identified {
		return false
	}
	return true
}