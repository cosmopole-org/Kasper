package secured

import (
	"kasper/src/abstract/models/core"
	model "kasper/src/core/module/actor/model/base"
	"log"
)

type Guard struct {
	IsUser    bool `json:"isUser"`
	IsInSpace bool `json:"isInSpace"`
	IsInTopic bool `json:"isInTopic"`
}

func (g *Guard) CheckValidity(app core.ICore, packet []byte, signature string, userId string, pointId string) (bool, *model.Info) {
	log.Println("hello 12.........")
	if !g.IsUser {
		return true, model.NewInfo("", "")
	}
	log.Println("hello 13.........")
	identified, _, isGod := app.Tools().Security().AuthWithSignature(userId, packet, signature)
	if !identified {
		return false, &model.Info{}
	}
	log.Println("hello 14.........")
	if !g.IsInSpace {
		return true, model.NewGodInfo(userId, "", isGod)
	}
	log.Println("hello 15.........")
	hasAccess := app.Tools().Security().HasAccessToPoint(userId, pointId)
	log.Println("hello 16.........")
	if !hasAccess {
		return false, &model.Info{}
	}
	return true, model.NewGodInfo(userId, pointId, isGod)
}

func (g *Guard) CheckIdentity(app core.ICore, packet []byte, signature string, userId string) bool {
	if !g.IsUser {
		return true
	}
	identified, _, _ := app.Tools().Security().AuthWithSignature(userId, packet, signature)
	return identified
}
