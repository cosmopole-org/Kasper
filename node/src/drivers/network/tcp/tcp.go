package tcp

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/trx"
	"kasper/src/shell/utils/crypto"
	"kasper/src/shell/utils/future"
	"log"
	"net"
	"sync"

	iaction "kasper/src/abstract/models/action"

	packetmodel "kasper/src/abstract/models/packet"

	cmap "github.com/orcaman/concurrent-map/v2"
)

type Socket struct {
	Lock   sync.Mutex
	Conn   net.Conn
	Buffer [][]byte
	Ack    bool
	app    core.ICore
}

type Tcp struct {
	app     core.ICore
	sockets *cmap.ConcurrentMap[string, *Socket]
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
	socket := &Socket{Buffer: [][]byte{}, Conn: conn, app: t.app}
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
}

func (t *Socket) writeResponse(requestId string, resCode int, response any, writeRaw bool) {

	b1 := []byte(requestId)
	b1Len := make([]byte, 4)
	binary.BigEndian.PutUint64(b1Len, uint64(len(b1)))

	b2 := make([]byte, 4)
	binary.BigEndian.PutUint64(b2, uint64(resCode))

	var b3 []byte
	if writeRaw {
		b3 = response.([]byte)
	} else {
		var err error
		b3, err = json.Marshal(response)
		if err != nil {
			log.Println(err)
			return
		}
	}
	b3Len := make([]byte, 4)
	binary.BigEndian.PutUint64(b3Len, uint64(len(b3)))

	packet := make([]byte, 1+len(b1Len)+len(b1)+len(b2)+len(b3Len)+len(b3))
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

	t.Lock.Lock()
	defer t.Lock.Unlock()

	t.Buffer = append(t.Buffer, packet)
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
	pointer := 0
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

	var lis *signaler.Listener
	if path == "packet_received" {
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
	if path == "authenticate" {
		success, _, _ := t.app.Tools().Security().AuthWithSignature(userId, payload, signature)
		if success {
			lis = t.connectListener(userId)
			var pointIds []string
			t.app.ModifyState(true, func(trx trx.ITrx) {
				pIds, err := trx.GetLinksList("memberof::"+userId+"::", -1, -1)
				if err != nil {
					log.Println(err)
					pointIds = []string{}
				} else {
					pointIds = pIds
				}
			})
			for _, pointId := range pointIds {
				t.app.Tools().Signaler().JoinGroup(pointId, userId)
			}
			t.writeResponse(packetId, 0, packetmodel.BuildErrorJson("authenticated"), false)
			oldQueueEndPack, _ := json.Marshal(packetmodel.ResponseSimpleMessage{Message: "old_queue_end"})
			t.app.Tools().Signaler().ListenToSingle(lis)
			lis.Signal(oldQueueEndPack)
		} else {
			t.writeResponse(packetId, 4, packetmodel.BuildErrorJson("authentication failed"), false)
		}
		return
	}

	action := t.app.Actor().FetchAction(path)
	if action == nil {
		t.writeResponse(packetId, 1, packetmodel.BuildErrorJson("action not found"), false)
		return
	}

	var err error
	input, err := action.(iaction.ISecureAction).ParseInput("tcp", payload)
	if err != nil {
		log.Println(err)
		t.writeResponse(packetId, 2, packetmodel.BuildErrorJson(err.Error()), false)
		return
	}

	log.Println("hi")

	statusCode, result, err := action.(iaction.ISecureAction).SecurelyAct(userId, packetId, payload, signature, input, t.Conn.RemoteAddr().String())
	if statusCode == 1 {
		t.handleResultOfFunc(packetId, result)
		return
	} else if err != nil {
		httpStatusCode := 3
		if statusCode == -1 {
			httpStatusCode = 4
		}
		t.writeResponse(packetId, httpStatusCode, packetmodel.BuildErrorJson(err.Error()), false)
	}
	t.writeResponse(packetId, 0, result, false)
}

func NewTcp(app core.ICore) *Tcp {
	m := cmap.New[*Socket]()
	return &Tcp{app: app, sockets: &m}
}
