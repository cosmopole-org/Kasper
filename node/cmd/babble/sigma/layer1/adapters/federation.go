package adapters

import (
	models "kasper/cmd/babble/sigma/layer1/model"
)

type IFederation interface {
	SendInFederation(destOrg string, packet models.OriginPacket)
}
