package network

type IFederation interface {
	SendFedRequest(destOrg string, requestId string, userId string, path string, payload []byte, signature string)
	SendFedResponse(destOrg string, requestId string, resCode int, res any)
	SendFedUpdate(destOrg string, key string, updatePack any, targetType string, targetIdVal string, exceptions []string)
	SendFedRequestByCallback(destOrg string, requestId string, userId string, path string, payload []byte, signature string, callback func([]byte, int, error))
}
