package network

import (
	models "kasper/src/shell/layer1/model"
)

type IFederation interface {
	SendInFederation(destOrg string, packet models.OriginPacket)
	SendInFederationPacketByCallback(destOrg string, packet models.OriginPacket, callback func([]byte, int, error))
	SendInFederationFileReqByCallback(destOrg string, packet models.OriginPacket, callback func(string, error))
	SendInFederationFileResByCallback(destOrg string, packet models.OriginPacket)
}
