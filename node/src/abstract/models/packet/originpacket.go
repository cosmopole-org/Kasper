package packet

type OriginPacket struct {
	Key        string
	UserId     string
	PointId    string
	RequestId  string
	Binary     []byte
	Signature  string
	IsResponse bool
	Exceptions []string
}
