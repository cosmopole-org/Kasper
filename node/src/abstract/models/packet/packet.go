package packet

type Packet struct {
	Origin string `json:"origin"`
	Data   string `json:"data"`
}

type LogPacket struct {
	Id      string
	PointId string
	UserId  string
	Data    string
}
