package outputs_interact

import model "kasper/src/shell/api/model"

type SendFriendRequestOutput struct {
	Interaction model.Interaction `json:"interaction"`
}
