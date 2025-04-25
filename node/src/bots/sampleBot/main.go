package hokmagent

import (
	"encoding/json"
	"fmt"
	"kasper/src/abstract/models/action"
	"kasper/src/abstract/models/core"
	inputs_points "kasper/src/shell/api/inputs/points"
	"kasper/src/shell/utils/crypto"
)

type HokmAgent struct {
	Core  core.ICore
	UserId string
}

func (h *HokmAgent) Install(c core.ICore, uid string) {
	h.UserId = uid
	h.Core = c
}

func (h *HokmAgent) OnSignal(input inputs_points.SignalInput) any {
	return map[string]any{}
}

func (h *HokmAgent) SendTopicPacket(typ string, pointId string, userId string, data any) {
	innerData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return
	}
	packet := inputs_points.SignalInput{Type: typ, PointId: pointId, UserId: userId, Data: string(innerData)}
	packetBinary, err := json.Marshal(packet)
	if err != nil {
		fmt.Println(err)
		return
	}
	h.Core.Actor().FetchAction("/points/signal").(action.ISecureAction).SecurelyAct(
		h.UserId,
		crypto.SecureUniqueString(),
		packetBinary,
		"#botsign",
		packet,
		"",
	)
}
