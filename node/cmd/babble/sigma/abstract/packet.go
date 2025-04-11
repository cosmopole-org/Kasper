package abstract

type ChainPacket struct {
	Type    string
	Meta    map[string]any
	Key     string
	Payload []byte
	Effects Effects
}

type Effects struct {
	DbUpdates    []Update      `json:"dbUpdates"`
	CacheUpdates []CacheUpdate `json:"cacheUpdates"`
}

type EmptyPayload struct{}

type Update struct {
	Data string `json:"data"`
}

type CacheUpdate struct {
	Typ string `json:"type"`
	Key string `json:"key"`
	Val string `json:"val"`
}
