package module_model

type Listener struct {
	Id      string
	Paused  bool
	DisTime int64
	Signal  func(any)
}

type GlobalListener struct {
	Signal func(string, any)
}

type JoinListener struct {
	Join  func(string, string)
	Leave func(string, string)
}
