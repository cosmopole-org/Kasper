package outputs_interact

import model "kasper/cmd/babble/sigma/api/model"

type SendFriendRequestOutput struct {
	Interaction model.Interaction `json:"interaction"`
}
