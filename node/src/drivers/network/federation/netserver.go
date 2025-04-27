package net_federation

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/models/core"
	"kasper/src/shell/utils/crypto"
	"kasper/src/shell/utils/future"
	"log"
	"net"
	"strings"
	"sync"

	packetmodel "kasper/src/abstract/models/packet"

	cmap "github.com/orcaman/concurrent-map/v2"
)

type Socket struct {
	Lock   sync.Mutex
	Conn   net.Conn
	Buffer [][]byte
	Ack    bool
	app    core.ICore
	server *Tcp
}

type FedApi func(socket *Socket, srcIp string, packet packetmodel.OriginPacket)

type Tcp struct {
	app     core.ICore
	bridge  FedApi
	sockets *cmap.ConcurrentMap[string, *Socket]
}

func (t *Tcp) InjectBridge(bridge FedApi) {
	t.bridge = bridge
}

func (t *Tcp) Listen(port int) {
	future.Async(func() {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			fmt.Println(err)
			return
		}
		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Println(err)
				continue
			}
			log.Println("new client connected")
			future.Async(func() { t.handleConnection(conn) }, false)
		}
	}, true)
}

func (t *Tcp) handleConnection(conn net.Conn) {
	connId := crypto.SecureUniqueString()
	defer func() {
		t.sockets.Remove(connId)
		conn.Close()
	}()
	socket := &Socket{Buffer: [][]byte{}, Conn: conn, app: t.app, Ack: true, server: t}
	t.sockets.Set(connId, socket)
	lenBuf := make([]byte, 4)
	buf := make([]byte, 1024)
	readCount := 0
	oldReadCount := 0
	for {
		_, err := conn.Read(lenBuf)
		if err != nil {
			fmt.Println(err)
			return
		}
		length := int(binary.BigEndian.Uint32(lenBuf))
		readData := make([]byte, length)
		for {
			readLength, err := conn.Read(buf)
			if err != nil {
				fmt.Println(err)
				return
			}
			oldReadCount = readCount
			readCount += readLength
			if readCount >= length {
				copy(readData[oldReadCount:], buf[:readLength-(readCount-length)])
				copy(buf[0:readCount-length], buf[readLength-(readCount-length):readLength])
				readCount = readLength - (readCount - length)
				log.Println("packet received")
				log.Println(string(readData))
				socket.processPacket(readData)
				readCount -= length
				break
			} else {
				copy(readData[oldReadCount:readCount], buf[:readLength])
			}
		}
	}
}

func (t *Socket) writeUpdate(updatePack any, writeRaw bool) {

	var b3 []byte
	if writeRaw {
		b3 = updatePack.([]byte)
	} else {
		var err error
		b3, err = json.Marshal(updatePack)
		if err != nil {
			log.Println(err)
			return
		}
	}
	b3Len := make([]byte, 4)
	binary.BigEndian.PutUint64(b3Len, uint64(len(b3)))

	packet := make([]byte, 1+len(b3Len)+len(b3))
	pointer := 1

	packet[0] = 0x01
	copy(packet[pointer:pointer+len(b3Len)], b3Len[:])
	pointer += len(b3Len)
	copy(packet[pointer:pointer+len(b3)], b3[:])
	pointer += len(b3)

	t.Lock.Lock()
	defer t.Lock.Unlock()

	t.Buffer = append(t.Buffer, packet)
	t.pushBuffer()
}

func (t *Socket) writeResponse(requestId string, resCode int, response any, writeRaw bool) {

	log.Println("preparing response...")

	b1 := []byte(requestId)
	b1Len := make([]byte, 4)
	binary.BigEndian.PutUint32(b1Len, uint32(len(b1)))

	b2 := make([]byte, 4)
	binary.BigEndian.PutUint32(b2, uint32(resCode))

	var b3 []byte
	var b4 []byte
	if writeRaw {
		b3 = response.([]byte)
	} else {
		var err error
		b3, err = json.Marshal(response)
		if err != nil {
			log.Println(err)
			return
		}
		b4 = []byte(t.app.SignPacket(b3))
	}
	b3Len := make([]byte, 4)
	binary.BigEndian.PutUint32(b3Len, uint32(len(b3)))
	b4Len := make([]byte, 4)
	binary.BigEndian.PutUint32(b4Len, uint32(len(b4)))

	packet := make([]byte, 1+len(b1Len)+len(b1)+len(b2)+len(b3Len)+len(b3)+len(b4Len)+len(b4))
	pointer := 1

	packet[0] = 0x02

	copy(packet[pointer:pointer+len(b1Len)], b1Len[:])
	pointer += len(b1Len)
	copy(packet[pointer:pointer+len(b1)], b1[:])
	pointer += len(b1)

	copy(packet[pointer:pointer+len(b2)], b2[:])
	pointer += len(b2)

	copy(packet[pointer:pointer+len(b3Len)], b3Len[:])
	pointer += len(b3Len)
	copy(packet[pointer:pointer+len(b3)], b3[:])
	pointer += len(b3)

	copy(packet[pointer:pointer+len(b4Len)], b4Len[:])
	pointer += len(b4Len)
	copy(packet[pointer:pointer+len(b4)], b4[:])
	pointer += len(b4)

	log.Println("appending to buffer...")

	t.Lock.Lock()
	defer t.Lock.Unlock()

	t.Buffer = append(t.Buffer, packet)
	t.pushBuffer()
}

func (t *Socket) pushBuffer() {
	log.Println("pushing buffer to client...", t.Ack, len(t.Buffer))
	if t.Ack {
		if len(t.Buffer) > 0 {
			t.Ack = false
			_, err := t.Conn.Write(t.Buffer[0])
			if err != nil {
				t.Ack = true
				log.Println(err)
			}
		}
	}
}

func (t *Socket) handleResultOfFunc(requestId string, result any) {
	switch result := result.(type) {
	case packetmodel.Command:
		if result.Value == "sendFile" {
			content, _ := t.app.Tools().File().ReadFileByPath(result.Data)
			t.writeResponse(requestId, 0, content, true)
		} else {
			t.writeResponse(requestId, 0, result, false)
		}
	default:
		t.writeResponse(requestId, 0, result, false)
	}
}

func (t *Socket) connectListener(uid string) *signaler.Listener {
	t.app.Tools().Signaler().Lock()
	defer t.app.Tools().Signaler().Unlock()
	lis, found := t.app.Tools().Signaler().Listeners().Get(uid)
	if !found {
		lis = &signaler.Listener{
			Id:      uid,
			Paused:  false,
			DisTime: 0,
			Signal: func(b any) {
				if b != nil {
					t.writeUpdate(b, false)
				}
				t.Lock.Lock()
				defer t.Lock.Unlock()
				if len(t.Buffer) > 0 {
					if t.Ack {
						t.Ack = false
						_, err := t.Conn.Write(t.Buffer[0])
						if err != nil {
							t.Ack = true
							log.Println(err)
						}
					}
				}
			},
		}
	}
	t.Ack = true
	return lis
}

func (t *Socket) processPacket(packet []byte) {
	if len(packet) == len([]byte("packet_received")) && string(packet) == "packet_received" {
		send := func() {
			t.Lock.Lock()
			defer t.Lock.Unlock()
			t.Ack = true
			if len(t.Buffer) > 0 {
				t.Buffer = t.Buffer[1:]
				if len(t.Buffer) > 0 {
					t.Ack = false
					_, err := t.Conn.Write(t.Buffer[0])
					if err != nil {
						t.Ack = true
						log.Println(err)
					}
				}
			}
		}
		send()
		return
	}
	typ := ""
	switch packet[0] {
	case 0x01:
		{
			typ = "update"
			break
		}
	case 0x02:
		{
			typ = "response"
			break
		}
	case 0x03:
		{
			typ = "request"
			break
		}
	}
	var pack packetmodel.OriginPacket
	pointer := 1
	if typ == "request" {
		signatureLength := int(binary.BigEndian.Uint32(packet[pointer : pointer+4]))
		log.Println("signature length:", signatureLength)
		pointer += 4
		signature := string(packet[pointer : pointer+signatureLength])
		pointer += signatureLength
		log.Println("signature:", signature)
		userIdLength := int(binary.BigEndian.Uint32(packet[pointer : pointer+4]))
		pointer += 4
		log.Println("userId length:", userIdLength)
		userId := string(packet[pointer : pointer+userIdLength])
		pointer += userIdLength
		log.Println("userId:", userId)
		pathLength := int(binary.BigEndian.Uint32(packet[pointer : pointer+4]))
		pointer += 4
		log.Println("path length:", pathLength)
		path := string(packet[pointer : pointer+pathLength])
		pointer += pathLength
		log.Println("path:", path)
		packetIdLength := int(binary.BigEndian.Uint32(packet[pointer : pointer+4]))
		pointer += 4
		log.Println("packetId length:", packetIdLength)
		packetId := string(packet[pointer : pointer+packetIdLength])
		pointer += packetIdLength
		log.Println("packetId:", packetId)
		payload := packet[pointer:]
		log.Println(string(payload))
		pack = packetmodel.OriginPacket{Type: typ, Key: path, UserId: userId, PointId: "", Binary: payload, Signature: signature, RequestId: packetId, Exceptions: []string{}}
	} else if typ == "response" {
		signatureLength := int(binary.BigEndian.Uint32(packet[pointer : pointer+4]))
		log.Println("signature length:", signatureLength)
		pointer += 4
		signature := string(packet[pointer : pointer+signatureLength])
		pointer += signatureLength
		log.Println("signature:", signature)
		packetIdLength := int(binary.BigEndian.Uint32(packet[pointer : pointer+4]))
		pointer += 4
		log.Println("packetId length:", packetIdLength)
		packetId := string(packet[pointer : pointer+packetIdLength])
		pointer += packetIdLength
		log.Println("packetId:", packetId)
		payload := packet[pointer:]
		log.Println(string(payload))
		pack = packetmodel.OriginPacket{Type: typ, Key: "", UserId: "", PointId: "", Binary: payload, Signature: signature, RequestId: packetId, Exceptions: []string{}}
	} else if typ == "update" {
		signatureLength := int(binary.BigEndian.Uint32(packet[pointer : pointer+4]))
		log.Println("signature length:", signatureLength)
		pointer += 4
		signature := string(packet[pointer : pointer+signatureLength])
		pointer += signatureLength
		log.Println("signature:", signature)
		pointIdLength := int(binary.BigEndian.Uint32(packet[pointer : pointer+4]))
		log.Println("pointId length:", pointIdLength)
		pointer += 4
		pointId := string(packet[pointer : pointer+pointIdLength])
		pointer += pointIdLength
		log.Println("pointId:", pointId)
		exceptionsLength := int(binary.BigEndian.Uint32(packet[pointer : pointer+4]))
		log.Println("execptions length:", signatureLength)
		pointer += 4
		exceptions := []string{}
		err := json.Unmarshal(packet[pointer:pointer+signatureLength], &exceptions)
		pointer += exceptionsLength
		if err != nil {
			log.Println(err)
			return
		}
		payload := packet[pointer:]
		log.Println(string(payload))
		pack = packetmodel.OriginPacket{Type: typ, Key: "", UserId: "", PointId: pointId, Binary: payload, Signature: signature, RequestId: "", Exceptions: exceptions}
	}

	t.server.bridge(t, strings.Split(t.Conn.RemoteAddr().String(), ":")[0], pack)
}

func NewTcp(app core.ICore) *Tcp {
	m := cmap.New[*Socket]()
	return &Tcp{app: app, sockets: &m}
}
