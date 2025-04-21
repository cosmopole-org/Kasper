package chain

import "kasper/src/abstract/models/update"

type ChainPacket struct {
	Type    string
	Meta    map[string]any
	Key     string
	Payload []byte
	Effects Effects
}

type Election struct {
	MyNum        string
	Participants map[string]bool
	Commits      map[string][]byte
	Reveals      map[string]string
}

type ChainCallback struct {
	Fn        func([]byte, int, error)
	Executors map[string]bool
	Responses map[string]string
}

type Effects struct {
	DbUpdates    []update.Update      `json:"dbUpdates"`
}
