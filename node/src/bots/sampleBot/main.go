package hokmagent

import (
	"encoding/json"
	"fmt"
	"kasper/src/abstract"
	inputs_topics "kasper/src/shell/api/inputs/topics"
	"kasper/src/bots/sampleBot/models"
	"kasper/src/shell/layer1/adapters"
	module_model "kasper/src/shell/layer2/model"
)

type HokmAgent struct {
	Core  abstract.ICore
	Store adapters.IStorage
	Token string
}

func (h *HokmAgent) Install(c abstract.ICore, t string) {
	h.Token = t
	h.Core = c
	h.Store = abstract.UseToolbox[*module_model.ToolboxL2](c.Get(2).Tools()).Storage()
}

func (h *HokmAgent) OnTopicSend(input models.Send) any {

	return map[string]any{}
}

func (h *HokmAgent) SendTopicPacket(typ string, spaceId string, topicId string, memberId string, recvId string, data any) {
	innerData, err2 := json.Marshal(data)
	if err2 != nil {
		fmt.Println(err2)
		return
	}
	packet := inputs_topics.SendInput{Type: typ, SpaceId: spaceId, TopicId: topicId, MemberId: memberId, RecvId: recvId, Data: string(innerData)}
	h.Core.Get(1).Actor().FetchAction("/topics/send").(abstract.ISecureAction).SecurelyAct(
		h.Core.Get(1),
		h.Token,
		"",
		packet,
		"",
	)
}
