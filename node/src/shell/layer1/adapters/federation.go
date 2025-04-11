package adapters

import (
	models "kasper/src/shell/layer1/model"
)

type IFederation interface {
	SendInFederation(destOrg string, packet models.OriginPacket)
}
