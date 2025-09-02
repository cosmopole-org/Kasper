package chain

import (
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/trx"
	"kasper/src/shell/api/model"
	"kasper/src/shell/utils/future"
	"log"
	"maps"
	"math"
	"math/rand"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
	queues "github.com/theodesp/blockingQueues"
)

var initialMap = map[string]int64{
	"api.decillionai.com": 1,
}

type Transaction struct {
	Typ     string
	Payload []byte
}

type Guarantee struct {
	Proof  string
	Sign   string
	RndNum int
}

type Event struct {
	Transactions    []Transaction
	Proof           string
	Origin          string
	MyUpdate        []byte
	Timestamp       int64
	backedResponses map[string]Guarantee
	randomNums      map[string]int
	backedProofs    map[string]string
	electionReadys  map[string]bool
	phase           int
}

type Packet struct {
	chainId int64
	data    []byte
}

type Socket struct {
	Lock       sync.Mutex
	Id         string
	Conn       net.Conn
	Buffer     []Packet
	Ack        bool
	app        core.ICore
	blockchain *Blockchain
}

type SubChain struct {
	Lock                  sync.Mutex
	id                    int64
	events                map[string]*Event
	pendingEvents         []*Event
	pendingBlockElections int
	readyForNewElection   bool
	cond_var_             chan int
	readyElectors         map[string]bool
	nextEventVotes        map[string]string
	nextEventQueue        *queues.BlockingQueue
	nextBlockQueue        *queues.BlockingQueue
	blocks                []*Event
	pendingTrxs           []Transaction
	peers                 map[string]int64
	chain                 *Chain
	removed               bool
	remainedCount         int
}

type Chain struct {
	id         int64
	blockchain *Blockchain
	SubChains  *cmap.ConcurrentMap[string, *SubChain]
	MyShards   *cmap.ConcurrentMap[string, bool]
	sharder    *DynamicShardingSystem
}

type Blockchain struct {
	app          core.ICore
	owners       *cmap.ConcurrentMap[string, *cmap.ConcurrentMap[string, bool]]
	origToOwner  *cmap.ConcurrentMap[string, string]
	sockets      *cmap.ConcurrentMap[string, *Socket]
	chains       *cmap.ConcurrentMap[string, *Chain]
	allSubChains *cmap.ConcurrentMap[string, *SubChain]
	pipeline     func([][]byte) []string
	chainCounter int64
}

type Ok struct {
	Proof  string
	RndNum int
}

func NewChain(core core.ICore) *Blockchain {
	m := cmap.New[*Socket]()
	q, _ := queues.NewLinkedBlockingQueue(1000)
	q2, _ := queues.NewLinkedBlockingQueue(1000)
	peers := map[string]int64{}
	maps.Copy(peers, initialMap)
	peers[core.Id()] = 1
	sc := &SubChain{
		id:                    1,
		events:                map[string]*Event{},
		pendingBlockElections: 0,
		readyForNewElection:   true,
		cond_var_:             make(chan int, 10000),
		readyElectors:         map[string]bool{},
		nextEventVotes:        map[string]string{},
		nextBlockQueue:        q,
		nextEventQueue:        q2,
		pendingTrxs:           []Transaction{},
		pendingEvents:         []*Event{},
		blocks:                []*Event{},
		remainedCount:         0,
		peers:                 peers,
	}
	m2 := cmap.New[*SubChain]()
	m2.Set("1", sc)
	m3 := cmap.New[bool]()
	m3.Set("1", true)
	chain := &Chain{
		SubChains: &m2,
		MyShards:  &m3,
	}
	chain.sharder = NewSharder(chain)
	sc.chain = chain
	m4 := cmap.New[*Chain]()
	m4.Set("1", chain)
	m5 := cmap.New[*SubChain]()
	m5.Set("1", sc)
	m6 := cmap.New[*cmap.ConcurrentMap[string, bool]]()
	m7 := cmap.New[string]()
	blockchain := &Blockchain{
		app:          core,
		owners:       &m6,
		origToOwner:  &m7,
		sockets:      &m,
		chainCounter: math.MaxInt32,
		chains:       &m4,
		pipeline:     nil,
		allSubChains: &m5,
	}
	chain.blockchain = blockchain
	return blockchain
}

func (t *SubChain) appendEvent(e *Event) {
	t.Lock.Lock()
	defer t.Lock.Unlock()
	t.events[e.Proof] = e
	t.pendingEvents = append(t.pendingEvents, e)
	t.remainedCount++
}

func (t *SubChain) SendToNode(origin string, payload []byte) {
	log.Println("sending packet to peer:", origin)
	s, ok := t.chain.blockchain.sockets.Get(origin)
	if ok {
		s.writePacket(t.id, payload)
	} else {
		log.Println("peer socket not found")
	}
}

func (t *SubChain) BroadcastInShard(payload []byte) {
	for k := range t.peers {
		if k == t.chain.blockchain.app.Id() {
			continue
		}
		s, ok := t.chain.blockchain.sockets.Get(k)
		if ok {
			s.writePacket(t.id, payload)
		}
	}
}

func (b *Blockchain) Listen(port int, tlsConfig *tls.Config) {
	for _, t := range b.chains.Items() {
		for _, subchain := range t.SubChains.Items() {
			future.Async(func() {
				subchain.Run()
			}, false)
		}
	}
	future.Async(func() {
		ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", port), tlsConfig)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		defer ln.Close()
		log.Println("Chains' TLS server listening on port ", port)

		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Println(err)
				continue
			}
			log.Println("new client connected")
			initialWord := []byte("I_WANNA_JOIN_NETWORK")
			question := make([]byte, len(initialWord))
			n, err := conn.Read(question)
			if err != nil {
				log.Println(err)
				continue
			}
			if (n != len(initialWord)) || (string(initialWord) != string(question)) {
				log.Println("notice: connected chain peer is not verified, skipping...")
				continue
			}
			b.handleConnection(conn, "")
		}
	}, true)
}

func (t *Blockchain) listenForPackets(socket *Socket) {
	defer func() {
		t.sockets.Remove(socket.Id)
		socket.Conn.Close()
	}()
	origin := strings.Split(socket.Conn.RemoteAddr().String(), ":")[0]
	lenBuf := make([]byte, 4)
	chainIdBuf := make([]byte, 8)
	buf := make([]byte, 1024)
	nextBuf := make([]byte, 2048)
	readCount := 0
	oldReadCount := 0
	enough := false
	beginning := true
	length := 0
	chainId := int64(0)
	readLength := 0
	remainedReadLength := 0
	var readData []byte
	for {
		if !enough {
			var err error
			readLength, err = socket.Conn.Read(buf)
			if err != nil {
				log.Println(origin, err)
				log.Println(origin, "socket had error and closed")
				return
			}
			func() {
				socket.Lock.Lock()
				defer socket.Lock.Unlock()
				log.Println(origin, "stat 0: reading data...")

				log.Println(origin, buf[0:readLength])

				readCount += readLength
				copy(nextBuf[remainedReadLength:remainedReadLength+readLength], buf[0:readLength])
				remainedReadLength += readLength

				log.Println(origin, nextBuf[0:readLength])

				log.Println(origin, "stat 1:", readLength, oldReadCount, readCount, remainedReadLength)
			}()
		}

		if beginning {
			if readCount >= 12 {
				func() {
					socket.Lock.Lock()
					defer socket.Lock.Unlock()
					log.Println(origin, "stating stat 2...")
					copy(lenBuf, nextBuf[0:4])
					log.Println(origin, "nextBuf", nextBuf[0:4])
					log.Println(origin, "lenBuf", lenBuf[0:4])
					remainedReadLength -= 4
					copy(nextBuf[0:remainedReadLength], nextBuf[4:remainedReadLength+4])
					length = int(binary.LittleEndian.Uint32(lenBuf))
					readData = make([]byte, length)
					readCount -= 4

					copy(chainIdBuf, nextBuf[0:8])
					log.Println(origin, "nextBuf", nextBuf[0:8])
					log.Println(origin, "lenBuf", chainIdBuf[0:8])
					remainedReadLength -= 8
					copy(nextBuf[0:remainedReadLength], nextBuf[8:remainedReadLength+8])
					chainId = int64(binary.LittleEndian.Uint64(chainIdBuf))
					readCount -= 8

					beginning = false
					enough = true

					log.Println(origin, "stat 2:", remainedReadLength, length, readCount)
				}()

			} else {
				enough = false
			}
		} else {
			if remainedReadLength == 0 {
				enough = false
			} else if readCount >= length {
				func() {
					socket.Lock.Lock()
					defer socket.Lock.Unlock()
					log.Println(origin, "stating stat 3...")
					log.Println(origin, "stat 3 step 1", oldReadCount, length)
					copy(readData[oldReadCount:length], nextBuf[0:length-oldReadCount])
					log.Println(origin, "stat 3 step 2", readLength, readCount, length)
					readCount -= length
					copy(nextBuf[0:readCount], nextBuf[length-oldReadCount:(length-oldReadCount)+readCount])
					log.Println(origin, "nextBuf", nextBuf[0:readCount])
					log.Println(origin, "stat 3 step 3", readCount, length)
					remainedReadLength = readCount
					log.Println(origin, "packet received")
					packet := make([]byte, length)
					copy(packet, readData)
					lengthOfPacket := length
					chainIdOfPacket := chainId
					log.Println(origin, "stat 3 step 4")
					oldReadCount = 0
					enough = true
					beginning = true

					log.Println(origin, "stat 3:", remainedReadLength, oldReadCount, readCount)

					future.Async(func() {
						socket.processPacket(fmt.Sprintf("%d", chainIdOfPacket), origin, packet, lengthOfPacket)
					}, false)
				}()
			} else {
				func() {
					socket.Lock.Lock()
					defer socket.Lock.Unlock()
					log.Println(origin, "stating stat 4...")

					copy(readData[oldReadCount:oldReadCount+(readCount-oldReadCount)], nextBuf[0:readCount-oldReadCount])
					remainedReadLength = 0
					oldReadCount = readCount
					enough = true

					log.Println(origin, "stat 4:", remainedReadLength)
				}()
			}
		}
	}
}

func (t *Blockchain) handleConnection(conn net.Conn, orig string) {
	origin := ""
	if orig == "" {
		t.app.ModifyState(true, func(trx trx.ITrx) error {
			addr := conn.RemoteAddr().String()
			log.Println(addr)
			origin = trx.GetLink("PendingNode::" + strings.Split(addr, ":")[0])
			return nil
		})
		if origin == "" {
			log.Println("pending node is not registered")
			return
		}
	} else {
		origin = orig
	}
	newNode := Node{ID: origin, Power: rand.Intn(100)}
	for _, c := range t.chains.Items() {
		c.sharder.HandleNewNode(newNode)
	}
	socket := &Socket{Id: strings.Split(conn.RemoteAddr().String(), ":")[0], Buffer: []Packet{}, Conn: conn, app: t.app, blockchain: t, Ack: true}
	t.sockets.Set(origin, socket)
	future.Async(func() {
		t.listenForPackets(socket)
	}, false)
}

func (t *Socket) writePacket(chainId int64, packet []byte) {

	log.Println("appending to buffer...")

	t.Lock.Lock()
	defer t.Lock.Unlock()

	t.Buffer = append(t.Buffer, Packet{chainId: chainId, data: packet})
	t.pushBuffer()
}

func (t *Socket) pushBuffer() {
	log.Println("pushing buffer to client...", t.Ack, len(t.Buffer))
	if t.Ack {
		if len(t.Buffer) > 0 {
			t.Ack = false
			packetLen := make([]byte, 4)
			binary.LittleEndian.PutUint32(packetLen, uint32(len(t.Buffer[0].data)))
			_, err := t.Conn.Write(packetLen)
			if err != nil {
				t.Ack = true
				log.Println(err)
				return
			}
			chainId := make([]byte, 8)
			binary.LittleEndian.PutUint64(chainId, uint64(t.Buffer[0].chainId))
			_, err = t.Conn.Write(chainId)
			if err != nil {
				t.Ack = true
				log.Println(err)
				return
			}
			_, err = t.Conn.Write(t.Buffer[0].data)
			if err != nil {
				t.Ack = true
				log.Println(err)
				return
			}
		}
	}
}

func (chainSocket *Socket) processPacket(chainId string, origin string, packet []byte, packetLength int) {

	log.Println(origin, "received packet length: ", packetLength, len(packet))

	if packetLength == 1 && packet[0] == 0x01 {
		send := func() {
			chainSocket.Lock.Lock()
			defer chainSocket.Lock.Unlock()
			chainSocket.Ack = true
			if len(chainSocket.Buffer) > 0 {
				chainSocket.Buffer = chainSocket.Buffer[1:]
				chainSocket.pushBuffer()
			}
		}
		send()
		return
	}

	var subchain *SubChain

	func() {
		chainSocket.Lock.Lock()
		defer chainSocket.Lock.Unlock()
		packetLen := make([]byte, 4)
		binary.LittleEndian.PutUint32(packetLen, uint32(1))
		_, err := chainSocket.Conn.Write(packetLen)
		if err != nil {
			log.Println(err)
		}
		_, err = chainSocket.Conn.Write([]byte{0x01})
		if err != nil {
			log.Println(err)
		}
		ok := false
		subchain, ok = chainSocket.blockchain.allSubChains.Get(chainId)
		if !ok {
			subchain = nil
		}
	}()

	if subchain == nil {
		log.Println("subchain not found")
		return
	}

	pointer := 1

	if packet[0] == 0x07 {

		signature := ""
		userId := ""

		tempBytes := make([]byte, 4)
		copy(tempBytes, packet[pointer:pointer+4])
		signatureLength := int(binary.LittleEndian.Uint32(tempBytes))
		log.Println("signature length: ", signatureLength)
		pointer += 4
		if signatureLength > 0 {
			signature = string(packet[pointer : pointer+signatureLength])
			pointer += signatureLength
		}
		log.Println("signature: ", signature)

		tempBytes = make([]byte, 4)
		copy(tempBytes, packet[pointer:pointer+4])
		dataLength := int(binary.LittleEndian.Uint32(tempBytes))
		log.Println("data length: ", dataLength)
		pointer += 4
		if dataLength > 0 {
			userId = string(packet[pointer : pointer+dataLength])
			pointer += dataLength
		}
		log.Println("ownerId: ", userId)

		if success, _, _ := chainSocket.app.Tools().Security().AuthWithSignature(userId, []byte(userId), signature); success {
			if !chainSocket.blockchain.owners.Has(userId) {
				m := cmap.New[bool]()
				chainSocket.blockchain.owners.Set(userId, &m)
			}
			owner, _ := chainSocket.blockchain.owners.Get(userId)
			owner.Set(origin, true)
			chainSocket.blockchain.origToOwner.Set(origin, userId)
		}
	} else if packet[0] == 0x01 {

		signature := ""
		data := ""

		log.Println("received consensus packet phase 1 from ", origin)

		tempBytes := make([]byte, 4)
		copy(tempBytes, packet[pointer:pointer+4])
		signatureLength := int(binary.LittleEndian.Uint32(tempBytes))
		log.Println("signature length: ", signatureLength)
		pointer += 4
		if signatureLength > 0 {
			signature = string(packet[pointer : pointer+signatureLength])
			pointer += signatureLength
		}
		log.Println("signature: ", signature)

		tempBytes = make([]byte, 4)
		copy(tempBytes, packet[pointer:pointer+4])
		dataLength := int(binary.LittleEndian.Uint32(tempBytes))
		log.Println("data length: ", dataLength)
		pointer += 4
		if dataLength > 0 {
			data = string(packet[pointer : pointer+dataLength])
			pointer += dataLength
		}
		log.Println("data: ", data)

		eventObj := &Event{}
		err := json.Unmarshal([]byte(data), eventObj)
		if err != nil {
			log.Println(err)
			return
		}
		eventObj.backedProofs = map[string]string{}
		eventObj.backedResponses = map[string]Guarantee{}
		eventObj.randomNums = map[string]int{}
		eventObj.electionReadys = map[string]bool{}
		eventObj.phase = 2

		log.Println("step 1")

		if eventObj.Origin != origin {
			log.Println("origins does not match :\n", eventObj.Origin, "\n", origin)
			return
		}

		log.Println("step 2")

		subchain.appendEvent(eventObj)
		proofSign := chainSocket.blockchain.app.SignPacket([]byte(eventObj.Proof))

		log.Println("step 3")

		proofSignBytes := []byte(proofSign)
		proofSignLenBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(proofSignLenBytes[:], uint32(len(proofSignBytes)))
		proofBytes := []byte(eventObj.Proof)
		proofLenBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(proofLenBytes[:], uint32(len(proofBytes)))

		log.Println("step 4")

		updateLen := 1 + 4 + len(proofSignBytes) + 4 + len(proofBytes)

		update := make([]byte, updateLen)
		pointer := 1
		update[0] = 0x04
		copy(update[pointer:pointer+4], proofSignLenBytes)
		pointer += 4
		copy(update[pointer:pointer+len(proofSignBytes)], proofSignBytes)
		pointer += len(proofSignBytes)
		copy(update[pointer:pointer+4], proofLenBytes)
		pointer += 4
		copy(update[pointer:pointer+len(proofBytes)], proofBytes)
		pointer += len(proofBytes)
		eventObj.MyUpdate = update

		log.Println("step 5")

		okObj := Ok{Proof: eventObj.Proof, RndNum: rand.Intn(chainSocket.blockchain.sockets.Count())}
		okBytes, _ := json.Marshal(okObj)
		okLenBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(okLenBytes[:], uint32(len(okBytes)))
		okSign := chainSocket.app.SignPacket(okBytes)
		okSignLenBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(okSignLenBytes[:], uint32(len(okSign)))

		responseLen := 1 + 4 + len(okSign) + 4 + len(okBytes)
		response := make([]byte, responseLen)
		response[0] = 0x02
		pointer = 1
		copy(response[pointer:pointer+4], okSignLenBytes)
		pointer += 4
		copy(response[pointer:pointer+len(okSign)], okSign)
		pointer += len(okSign)
		copy(response[pointer:pointer+4], okLenBytes)
		pointer += 4
		copy(response[pointer:pointer+len(okBytes)], okBytes)
		pointer += len(okBytes)

		log.Println("step 6")

		subchain.SendToNode(origin, response)

	} else if packet[0] == 0x02 {

		sign := ""
		proof := ""

		log.Println("received consensus packet phase 2 from ", origin)

		tempBytes := make([]byte, 4)
		copy(tempBytes, packet[pointer:pointer+4])
		signLength := int(binary.LittleEndian.Uint32(tempBytes))
		log.Println("sign length: ", signLength)
		pointer += 4
		if signLength > 0 {
			sign = string(packet[pointer : pointer+signLength])
			pointer += signLength
		}
		log.Println("sign: ", sign)

		tempBytes2 := make([]byte, 4)
		copy(tempBytes2, packet[pointer:pointer+4])
		dataLength := int(binary.LittleEndian.Uint32(tempBytes2))
		log.Println("data length: ", dataLength)
		pointer += 4
		if dataLength > 0 {
			proof = string(packet[pointer : pointer+dataLength])
			pointer += dataLength
		}
		log.Println("ok: ", proof)

		okObj := Ok{}
		err := json.Unmarshal([]byte(proof), &okObj)
		if err != nil {
			log.Println(err)
			return
		}

		done := subchain.MemorizeResponseBacked(okObj.Proof, sign, okObj.RndNum, origin)

		if !done {
			return
		}

		e := subchain.GetEventByProof(okObj.Proof)
		e.phase = 2
		proofs, _ := json.Marshal(e.backedResponses)

		proofBytes := []byte(proofs)
		proofLenBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(proofLenBytes, uint32(len(proofBytes)))

		reqLen := 1 + 4 + len(proofBytes)
		req := make([]byte, reqLen)
		pointer = 1
		req[0] = 0x03
		copy(req[pointer:pointer+4], proofLenBytes)
		pointer += 4
		copy(req[pointer:pointer+len(proofBytes)], proofBytes)
		pointer += len(proofBytes)

		subchain.BroadcastInShard(req)

		proofBytes = []byte(okObj.Proof)
		proofLenBytes = make([]byte, 4)
		binary.LittleEndian.PutUint32(proofLenBytes, uint32(len(proofBytes)))

		reqLen = 1 + 4 + len(proofBytes)
		req = make([]byte, reqLen)
		pointer = 1
		req[0] = 0x06
		copy(req[pointer:pointer+4], proofLenBytes)
		pointer += 4
		copy(req[pointer:pointer+len(proofBytes)], proofBytes)
		pointer += len(proofBytes)

		subchain.BroadcastInShard(req)

	} else if packet[0] == 0x03 {

		proof := ""

		log.Println("received consensus packet phase 3 from ", origin)

		tempBytes2 := make([]byte, 4)
		copy(tempBytes2, packet[pointer:pointer+4])
		dataLength := int(binary.LittleEndian.Uint32(tempBytes2))
		log.Println("data length: ", dataLength)
		pointer += 4
		if dataLength > 0 {
			proof = string(packet[pointer : pointer+dataLength])
			pointer += dataLength
		}
		log.Println("proof: ", proof)

		m := map[string]Guarantee{}
		err := json.Unmarshal([]byte(proof), &m)
		if err != nil {
			log.Println(err)
			return
		}

		proofVal := m[chainSocket.app.Id()].Proof

		func() {
			chainSocket.Lock.Lock()
			defer chainSocket.Lock.Unlock()
			e := subchain.GetEventByProof(proofVal)
			rndNums := map[string]int{}
			for k, v := range m {
				rndNums[k] = v.RndNum
			}
			e.randomNums = rndNums
			e.phase = 3
			e.electionReadys[chainSocket.app.Id()] = true
		}()

		proofBytes := []byte(proofVal)
		proofLenBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(proofLenBytes, uint32(len(proofBytes)))

		reqLen := 1 + 4 + len(proofBytes)
		req := make([]byte, reqLen)
		pointer = 1
		req[0] = 0x06
		copy(req[pointer:pointer+4], proofLenBytes)
		pointer += 4
		copy(req[pointer:pointer+len(proofBytes)], proofBytes)
		pointer += len(proofBytes)

		subchain.BroadcastInShard(req)

	} else if packet[0] == 0x06 {

		log.Println("received consensus packet phase 3.1 from ", origin)

		proof := ""

		tempBytes2 := make([]byte, 4)
		copy(tempBytes2, packet[pointer:pointer+4])
		dataLength := int(binary.LittleEndian.Uint32(tempBytes2))
		log.Println("data length: ", dataLength)
		pointer += 4
		if dataLength > 0 {
			proof = string(packet[pointer : pointer+dataLength])
			pointer += dataLength
		}
		log.Println("proof: ", proof)

		readyToStart := false
		var e *Event
		func() {
			subchain.Lock.Lock()
			defer subchain.Lock.Unlock()
			e = subchain.GetEventByProof(proof)
			if e == nil {
				e = subchain.GetEventByProof(proof)
			}
			if _, ok := subchain.peers[origin]; ok {
				e.electionReadys[origin] = true
			}
			if len(e.electionReadys) == len(subchain.peers) {
				readyToStart = true
			}
		}()

		if readyToStart {
			subchain.nextEventQueue.Put(e)
		}

	} else if packet[0] == 0x04 {

		log.Println("received consensus packet phase 4 from ", origin)

		signature := ""
		vote := ""

		tempBytes := make([]byte, 4)
		copy(tempBytes, packet[pointer:pointer+4])
		pointer += 4
		signLength := int(binary.LittleEndian.Uint32(tempBytes))
		log.Println("signature length: ", signLength)
		if signLength > 0 {
			signature = string(packet[pointer : pointer+signLength])
			pointer += signLength
		}
		log.Println("signature: ", signature)

		tempBytes2 := make([]byte, 4)
		copy(tempBytes2, packet[pointer:pointer+4])
		pointer += 4
		dataLength := int(binary.LittleEndian.Uint32(tempBytes2))
		log.Println("data length: ", dataLength)
		if dataLength > 0 {
			vote = string(packet[pointer : pointer+dataLength])
			pointer += dataLength
		}
		log.Println("vote: ", vote)
		subchain.VoteForNextEvent(origin, vote)
	} else if packet[0] == 0x05 {
		log.Println("received consensus packet phase 5 from ", origin)
		subchain.NotifyElectorReady(origin)
	} else if packet[0] == 0xa1 {
		var data []byte
		tempBytes2 := make([]byte, 4)
		copy(tempBytes2, packet[pointer:pointer+4])
		pointer += 4
		dataLength := int(binary.LittleEndian.Uint32(tempBytes2))
		log.Println("data length: ", dataLength)
		if dataLength > 0 {
			data = packet[pointer : pointer+dataLength]
			pointer += dataLength
		}
		events := []*Event{}
		e := json.Unmarshal(data, &events)
		if e != nil {
			log.Println(e)
			return
		}
		for _, e := range events {
			for _, trx := range e.Transactions {
				if trx.Typ == "response" {
					chainSocket.blockchain.pipeline([][]byte{trx.Payload})
				}
			}
		}
	}
}

func (c *Blockchain) SetNodeStake(ownerId string, chainId int64, amount int64) {
	chain, ok := c.allSubChains.Get(fmt.Sprintf("%d", chainId))
	if !ok {
		log.Println("chain not found")
		return
	}
	if _, ok := chain.peers[ownerId]; ok {
		chain.peers[ownerId] = amount
	}
}

func (c *Blockchain) NotifyNewMachineCreated(chainId int64, machineId string) {
	chain, ok := c.chains.Get(fmt.Sprintf("%d", chainId))
	if !ok {
		log.Println("chain not found")
		return
	}
	sc := SmartContract{ID: machineId, TransactionCount: rand.Int63n(5)}
	chain.sharder.AssignContract(sc)
}

func (c *SubChain) NotifyElectorReady(origin string) {
	push := false
	func() {
		c.Lock.Lock()
		defer c.Lock.Unlock()
		c.readyElectors[origin] = true
		if len(c.readyElectors) == len(c.peers) {
			c.readyElectors = map[string]bool{}
			push = true
		}
	}()
	if push {
		c.cond_var_ <- 1
	}
}

func (c *SubChain) VoteForNextEvent(origin string, eventProof string) {

	var choosenEvent *Event = nil
	done := false
	func() {
		c.Lock.Lock()
		defer c.Lock.Unlock()
		if _, ok := c.peers[origin]; ok {
			c.nextEventVotes[origin] = eventProof
		}
		if len(c.nextEventVotes) == len(c.peers) {
			votes := map[string]int64{}
			for origin, vote := range c.nextEventVotes {
				if _, ok := votes[vote]; !ok {
					votes[vote] = c.peers[origin]
				} else {
					votes[vote] = votes[vote] + c.peers[origin]
				}
			}
			type Candidate struct {
				Proof string
				Votes int64
			}
			sortedArr := []Candidate{}
			for proof, votes := range votes {
				sortedArr = append(sortedArr, Candidate{proof, votes})
			}
			slices.SortStableFunc(sortedArr, func(a Candidate, b Candidate) int {
				if a.Votes > b.Votes {
					return 1
				} else if a.Votes < b.Votes {
					return -1
				} else {
					return 0
				}
			})

			if len(sortedArr) > 1 && (sortedArr[0].Votes == sortedArr[1].Votes) {
				topOnes := []Candidate{}
				sum := 0
				for _, cand := range sortedArr {
					if len(topOnes) < 2 {
						for _, v := range c.GetEventByProof(cand.Proof).randomNums {
							sum ^= v
						}
						topOnes = append(topOnes, cand)
					} else {
						if topOnes[0].Votes > cand.Votes {
							break
						} else {
							for _, v := range c.GetEventByProof(cand.Proof).randomNums {
								sum ^= v
							}
							topOnes = append(topOnes, cand)
						}
					}
				}
				result := sum % len(topOnes)
				type Cand struct {
					candVal Candidate
					num     uint32
				}
				m2 := []Cand{}
				for _, cand := range topOnes {
					orig := c.GetEventByProof(cand.Proof).Origin
					h := fnv.New32a()
					h.Write([]byte(orig))
					m2 = append(m2, Cand{num: h.Sum32() % uint32(len(topOnes)), candVal: cand})
				}
				slices.SortStableFunc(m2, func(a Cand, b Cand) int { return int(a.num) - int(b.num) })
				chosenProofRes := m2[0].candVal.Proof
				for _, cn := range m2 {
					if cn.num < uint32(result) {
						break
					}
					chosenProofRes = cn.candVal.Proof
				}
				choosenEvent = c.GetEventByProof(chosenProofRes)
			} else {
				choosenEvent = c.GetEventByProof(sortedArr[0].Proof)
			}
			done = true
		}
	}()
	if done {
		func() {
			c.Lock.Lock()
			defer c.Lock.Unlock()
			eventIndex := 0
			for _, event := range c.pendingEvents {
				if event.Proof == choosenEvent.Proof {
					break
				}
				eventIndex++
			}
			log.Println("chosen proof : ", choosenEvent.Proof)
			if eventIndex < len(c.pendingEvents) {
				c.pendingEvents = slices.Delete(c.pendingEvents, eventIndex, eventIndex+1)
			}
		}()

		c.nextBlockQueue.Put(choosenEvent)
		startNewElectionSignal := make([]byte, 1)
		startNewElectionSignal[0] = 0x05
		c.nextEventVotes = map[string]string{}

		c.BroadcastInShard(startNewElectionSignal)

		c.NotifyElectorReady(c.chain.blockchain.app.Id())
	}
}

func (c *SubChain) GetEventByProof(proof string) *Event {
	return c.events[proof]
}

func openSocket(origin string, chain *Blockchain) bool {
	log.Println("connecting to chain socket server: ", origin)
	addr := origin + ":8079"
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         origin,
	}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	log.Println("connected to the server..")
	conn.Write([]byte("I_WANNA_JOIN_NETWORK"))
	chain.handleConnection(conn, origin)
	return true
}

func (b *Blockchain) CreateWorkChain(peers map[string]int64) int64 {

	q, _ := queues.NewLinkedBlockingQueue(1000)
	q2, _ := queues.NewLinkedBlockingQueue(1000)
	b.chainCounter++
	sc := &SubChain{
		id:                    b.chainCounter,
		events:                map[string]*Event{},
		pendingBlockElections: 0,
		readyForNewElection:   true,
		cond_var_:             make(chan int, 10000),
		readyElectors:         map[string]bool{},
		nextEventVotes:        map[string]string{},
		nextBlockQueue:        q,
		nextEventQueue:        q2,
		pendingTrxs:           []Transaction{},
		pendingEvents:         []*Event{},
		blocks:                []*Event{},
		remainedCount:         0,
		peers:                 peers,
	}
	m2 := cmap.New[*SubChain]()
	m2.Set(fmt.Sprintf("%d", sc.id), sc)
	m3 := cmap.New[bool]()
	m3.Set(fmt.Sprintf("%d", sc.id), true)
	chain := &Chain{
		id:         sc.id,
		SubChains:  &m2,
		MyShards:   &m3,
		blockchain: b,
	}
	chain.sharder = NewSharder(chain)
	sc.chain = chain
	b.allSubChains.Set(fmt.Sprintf("%d", sc.id), sc)
	b.chains.Set(fmt.Sprintf("%d", chain.id), chain)
	return chain.id
}

func (b *Blockchain) UserOwnsOrigin(userId string, origin string) bool {
	ownerOrigins, ok := b.owners.Get(userId)
	if ok {
		return ownerOrigins.Has(origin)
	} else {
		return false
	}
}

func (b *Blockchain) GetNodeOwnerId(origin string) string {
	userId, ok := b.origToOwner.Get(origin)
	if ok {
		return userId
	} else {
		return ""
	}
}

func (b *Blockchain) GetValidatorsOfMachineShard(machineId string) []string {
	result := []string{}
	b.app.ModifyState(true, func(trx trx.ITrx) error {
		vm := model.Vm{MachineId: machineId}.Pull(trx)
		app := model.App{Id: vm.AppId}.Pull(trx)
		c, _ := b.chains.Get(fmt.Sprintf("%d", app.ChainId))
		mac, _ := c.sharder.contracts.Get(machineId)
		sc, _ := c.SubChains.Get(fmt.Sprintf("%d", mac.ShardID))
		for k := range sc.peers {
			result = append(result, k)
		}
		return nil
	})
	return result
}

func (c *Chain) CreateShardChain(subchainId int64) {
	q, _ := queues.NewLinkedBlockingQueue(1000)
	q2, _ := queues.NewLinkedBlockingQueue(1000)
	sc := &SubChain{
		chain:                 c,
		id:                    subchainId,
		events:                map[string]*Event{},
		pendingBlockElections: 0,
		readyForNewElection:   true,
		cond_var_:             make(chan int, 10000),
		readyElectors:         map[string]bool{},
		nextEventVotes:        map[string]string{},
		nextBlockQueue:        q,
		nextEventQueue:        q2,
		pendingTrxs:           []Transaction{},
		pendingEvents:         []*Event{},
		blocks:                []*Event{},
		remainedCount:         0,
		peers:                 map[string]int64{},
	}
	c.blockchain.allSubChains.Set(fmt.Sprintf("%d", sc.id), sc)
}

func (b *Blockchain) CreateTempChain(peers map[string]int64) int64 {
	q, _ := queues.NewLinkedBlockingQueue(1000)
	q2, _ := queues.NewLinkedBlockingQueue(1000)
	b.chainCounter++
	sc := &SubChain{
		id:                    b.chainCounter,
		events:                map[string]*Event{},
		pendingBlockElections: 0,
		readyForNewElection:   true,
		cond_var_:             make(chan int, 10000),
		readyElectors:         map[string]bool{},
		nextEventVotes:        map[string]string{},
		nextBlockQueue:        q,
		nextEventQueue:        q2,
		pendingTrxs:           []Transaction{},
		pendingEvents:         []*Event{},
		blocks:                []*Event{},
		remainedCount:         0,
		peers:                 peers,
	}
	m2 := cmap.New[*SubChain]()
	m2.Set(fmt.Sprintf("%d", sc.id), sc)
	m3 := cmap.New[bool]()
	m3.Set(fmt.Sprintf("%d", sc.id), true)
	chain := &Chain{
		id:         sc.id,
		SubChains:  &m2,
		MyShards:   &m3,
		blockchain: b,
	}
	chain.sharder = NewSharder(chain)
	sc.chain = chain
	b.allSubChains.Set(fmt.Sprintf("%d", sc.id), sc)
	b.chains.Set(fmt.Sprintf("%d", chain.id), chain)
	return chain.id
}

func (c *SubChain) Run() {

	future.Async(func() {
		for {
			haveTrxs := false
			if func() bool {
				c.Lock.Lock()
				defer c.Lock.Unlock()
				if c.removed {
					return true
				}
				if len(c.pendingTrxs) > 0 {
					haveTrxs = true
				}
				return false
			}() {
				for {
					if c.remainedCount == 0 {
						break
					}
					time.Sleep(time.Duration(100) * time.Millisecond)
				}
				c.chain.sharder.DoPostMerge(c.id)
				break
			}
			if haveTrxs {
				log.Println("creating new event...")
				var e *Event = nil
				if len(c.peers) == 1 {
					func() {
						c.Lock.Lock()
						defer c.Lock.Unlock()
						now := time.Now().UnixMicro()
						proof := fmt.Sprintf("%s-%d", c.chain.blockchain.app.Id(), now)
						e = &Event{
							Origin:          c.chain.blockchain.app.Id(),
							backedResponses: map[string]Guarantee{},
							backedProofs:    map[string]string{},
							randomNums:      map[string]int{},
							electionReadys:  map[string]bool{},
							Transactions:    c.pendingTrxs,
							Proof:           proof,
							MyUpdate:        []byte{},
							Timestamp:       now,
							phase:           6,
						}
						c.pendingTrxs = []Transaction{}
						c.events[e.Proof] = e
					}()
					c.nextBlockQueue.Put(e)
					continue
				} else {
					func() {
						c.Lock.Lock()
						defer c.Lock.Unlock()
						now := time.Now().UnixMicro()
						proof := fmt.Sprintf("%s-%d", c.chain.blockchain.app.Id(), now)
						e = &Event{
							Origin:          c.chain.blockchain.app.Id(),
							backedResponses: map[string]Guarantee{},
							backedProofs:    map[string]string{},
							randomNums:      map[string]int{},
							electionReadys:  map[string]bool{},
							Transactions:    c.pendingTrxs,
							Proof:           proof,
							MyUpdate:        []byte{},
							Timestamp:       now,
							phase:           1,
						}
						c.pendingTrxs = []Transaction{}
						c.events[e.Proof] = e
						c.pendingEvents = append(c.pendingEvents, e)
						c.remainedCount++
					}()
				}
				dataStr, _ := json.Marshal(e)
				signature := c.chain.blockchain.app.SignPacket(dataStr)
				dataBytes := []byte(dataStr)
				dataLenBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(dataLenBytes, uint32(len(dataBytes)))
				signBytes := []byte(signature)
				signLenBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(signLenBytes, uint32(len(signBytes)))

				payloadLen := 1 + 4 + len(signBytes) + 4 + len(dataBytes)
				payload := make([]byte, payloadLen)
				pointer := uint32(1)
				payload[0] = 0x01
				copy(payload[pointer:pointer+4], signLenBytes)
				pointer += 4
				copy(payload[pointer:pointer+uint32(len(signBytes))], signBytes)
				pointer += uint32(len(signBytes))
				copy(payload[pointer:pointer+4], dataLenBytes)
				pointer += 4
				copy(payload[pointer:pointer+uint32(len(dataBytes))], dataBytes)
				pointer += uint32(len(dataBytes))

				proofSign := c.chain.blockchain.app.SignPacket([]byte(e.Proof))
				proofSignBytes := []byte(proofSign)
				proofSignLenBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(proofSignLenBytes, uint32(len(proofSignBytes)))
				proofBytes := []byte(e.Proof)
				proofLenBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(proofLenBytes, uint32(len(proofBytes)))

				updateLen := 1 + 4 + len(proofSignBytes) + 4 + len(proofBytes)
				update := make([]byte, updateLen)
				pointer = 1
				update[0] = 0x04
				copy(update[pointer:pointer+4], proofSignLenBytes)
				pointer += 4
				copy(update[pointer:pointer+uint32(len(proofSignBytes))], proofSignBytes)
				pointer += uint32(len(proofSignBytes))
				copy(update[pointer:pointer+4], proofLenBytes)
				pointer += 4
				copy(update[pointer:pointer+uint32(len(proofBytes))], proofBytes)
				pointer += uint32(len(proofBytes))
				e.MyUpdate = update

				c.BroadcastInShard(payload)
			}
			time.Sleep(time.Duration(100) * time.Millisecond)
		}
	}, false)

	future.Async(func() {
		for {
			eRaw, err := c.nextEventQueue.Get()
			if err != nil {
				log.Println(err)
				panic(err)
			}
			e := eRaw.(*Event)
			allowed := false
			func() {
				if c.readyForNewElection {
					c.readyForNewElection = false
					allowed = true
				}
			}()
			if allowed {
				c.cond_var_ <- 1
			}
			<-c.cond_var_
			c.VoteForNextEvent(c.chain.blockchain.app.Id(), e.Proof)
			c.BroadcastInShard(e.MyUpdate)
		}
	}, true)

	future.Async(func() {
		for {
			blockRaw, _ := c.nextBlockQueue.Get()
			block := blockRaw.(*Event)
			c.blocks = append(c.blocks, block)
			pipelinePacket := [][]byte{}
			foundFunctionalTrx := false
			for _, trx := range block.Transactions {
				log.Println("received transaction: ", trx.Typ, string(trx.Payload))
				if c.id == 1 {
					if trx.Typ == "logLoad" {
						machineIds := []string{}
						e := json.Unmarshal(trx.Payload, &machineIds)
						if e != nil {
							log.Println(e)
						} else {
							for _, macId := range machineIds {
								c.chain.sharder.LogLoad(c.id, macId)
							}
						}
					} else if trx.Typ == "newNode" {
						origin := string(trx.Payload)
						c.chain.blockchain.pendingNode(origin)
					} else if trx.Typ == "joinWorkchain" {
						chainId := int64(binary.LittleEndian.Uint64(trx.Payload[0:8]))
						chain, ok := c.chain.blockchain.chains.Get(fmt.Sprintf("%d", chainId))
						if ok {
							newNode := Node{ID: string(trx.Payload[8:]), Power: rand.Intn(100)}
							chain.sharder.HandleNewNode(newNode)
						}
					} else {
						foundFunctionalTrx = true
						pipelinePacket = append(pipelinePacket, trx.Payload)
					}
				}
			}
			if foundFunctionalTrx {
				machineIds := c.chain.blockchain.pipeline(pipelinePacket)
				midsBytes, _ := json.Marshal(machineIds)
				sc, _ := c.chain.SubChains.Get("1")
				sc.SubmitTrx("logLoad", midsBytes)
			}
			func() {
				c.Lock.Lock()
				defer c.Lock.Unlock()
				c.remainedCount--
			}()
		}
	}, false)

	future.Async(func() {
		log.Println("trying to connect to other peers...")
		peers := map[string]int64{}
		maps.Copy(peers, initialMap)
		peers[c.chain.blockchain.app.Id()] = 1
		peersArr := []string{}
		for k, _ := range peers {
			peersArr = append(peersArr, k)
		}
		completed := false
		for !completed {
			completed = true
			for _, peerAddress := range peersArr {
				log.Println("socket: ", peerAddress)
				if peerAddress == c.chain.blockchain.app.Id() {
					c.chain.blockchain.sockets.Set(peerAddress, &Socket{app: c.chain.blockchain.app, blockchain: c.chain.blockchain, Conn: nil, Buffer: []Packet{}, Ack: true})
					continue
				}
				// if peerAddress < c.chain.blockchain.app.Id() {
				// 	continue
				// }
				if c.chain.blockchain.sockets.Has(peerAddress) {
					continue
				}
				if !openSocket(peerAddress, c.chain.blockchain) {
					completed = false
					continue
				}
			}
			time.Sleep(time.Duration(1) * time.Second)
		}
	}, false)
}

func (b *Blockchain) pendingNode(origin string) {
	ips, _ := net.LookupIP(origin)
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			address := ipv4.String()
			b.app.ModifyState(false, func(trx trx.ITrx) error {
				trx.PutLink("PendingNode::"+address, origin)
				return nil
			})
			break
		}
	}
}

func (c *SubChain) SubmitTrx(typ string, payload []byte) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.pendingTrxs = append(c.pendingTrxs, Transaction{Typ: typ, Payload: payload})
}

func (b *Blockchain) SubmitTrx(chainId string, machineId string, typ string, payload []byte) {
	if chainId != "" {
		if machineId != "" {
			c, ok := b.chains.Get(chainId)
			if !ok {
				log.Println("submitting to subchain failed as chain not found")
				return
			}
			contract, ok := c.sharder.contracts.Get(machineId)
			if !ok {
				log.Println("submitting to subchain failed as contract not found")
				return
			}
			subchain, ok := c.SubChains.Get(fmt.Sprintf("%d", contract.ShardID))
			if !ok {
				log.Println("submitting to subchain failed as subchain of contract not found")
				return
			}
			subchain.SubmitTrx(typ, payload)
		} else {
			c, ok := b.chains.Get(chainId)
			if !ok {
				log.Println("submitting to subchain failed as chain not found")
				return
			}
			subchain, ok := c.SubChains.Get(chainId)
			if !ok {
				log.Println("submitting to subchain failed as subchain not found")
				return
			}
			subchain.SubmitTrx(typ, payload)
		}
	}
}

func (c *SubChain) MemorizeResponseBacked(proof string, signature string, rndNum int, origin string) bool {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	if e, ok := c.events[proof]; ok {
		e.backedResponses[origin] = Guarantee{Proof: proof, Sign: signature, RndNum: rndNum}
		if c.chain.MyShards.Has(c.chain.blockchain.app.Id()) {
			if len(e.backedResponses) == (len(c.peers) - 1) {
				return true
			}
		} else {
			if len(e.backedResponses) == len(c.peers) {
				return true
			}
		}
	} else {
		panic("event proof not found")
	}
	return false
}

func (c *SubChain) AddBackedProof(proof string, origin string, signedProof string) bool {
	// EVP_PKEY *pkey;
	// {
	//     std::lock_guard<std::mutex> lock(this->lock);
	//     pkey = this->shardPeers[origin]->pkey;
	// }
	// if (Utils::getInstance().verify_signature_rsa(pkey, proof, signedProof))
	if true {
		c.Lock.Lock()
		defer c.Lock.Unlock()
		if e, ok := c.events[proof]; ok {
			e.backedProofs[origin] = signedProof
			if len(e.backedProofs) == (len(c.peers) - 2) {
				if _, ok = e.backedProofs[e.Origin]; !ok {
					return true
				}
			}
		}
	}
	return false
}

func (c *Blockchain) RemoveConnection(origin string) {
	c.sockets.Remove(origin)
}

func (c *Blockchain) RegisterPipeline(pipeline func([][]byte) []string) {
	c.pipeline = pipeline
}

func (c *Blockchain) Peers() []string {
	return c.sockets.Keys()
}
