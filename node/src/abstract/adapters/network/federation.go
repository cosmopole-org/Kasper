package network

import "kasper/src/abstract/models/packet"

type IFederation interface {
	SendInFederation(destOrg string, pack packet.OriginPacket)
	SendInFederationPacketByCallback(destOrg string, packet packet.OriginPacket, callback func([]byte, int, error))
	SendInFederationFileReqByCallback(destOrg string, fileId string, packet packet.OriginPacket, callback func(string, error))
	SendInFederationFileResByCallback(destOrg string, packet packet.OriginFileRes)
}
