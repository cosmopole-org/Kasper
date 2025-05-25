package chain

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"kasper/src/abstract/models/core"
	"kasper/src/shell/utils/crypto"
	"kasper/src/shell/utils/future"
	"log"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
	queues "github.com/theodesp/blockingQueues"
)

type Transaction struct {
	Typ     string
	Payload []byte
}

type Event struct {
	Transactions    []Transaction
	Proof           string
	Origin          string
	MyUpdate        []byte
	backedResponses map[string]bool
	backedProofs    map[string]string
}

type Socket struct {
	Lock   sync.Mutex
	Id     string
	Conn   net.Conn
	Buffer [][]byte
	Ack    bool
	app    core.ICore
	chain  *Chain
}

type Chain struct {
	Lock                  sync.Mutex
	app                   core.ICore
	sockets               *cmap.ConcurrentMap[string, *Socket]
	proofEvents           map[string]*Event
	pendingEvents         []*Event
	chosenProof           string
	pendingBlockElections int
	readyForNewElection   bool
	ready                 bool
	cond_var_             chan int
	readyElectors         map[string]bool
	nextEventVotes        map[string]string
	nextBlockQueue        *queues.BlockingQueue
	pendingTrxs           []Transaction
}

func NewChain(core core.ICore) *Chain {
	m := cmap.New[*Socket]()
	q, _ := queues.NewLinkedBlockingQueue(1000)
	return &Chain{
		app: core,
		sockets: &m,
		proofEvents: map[string]*Event{},
		pendingEvents: []*Event{},
		chosenProof: "",
		pendingBlockElections: 0,
		readyForNewElection: true,
		ready: false,
		cond_var_: make(chan int),
		readyElectors: map[string]bool{},
		nextEventVotes: map[string]string{},
		nextBlockQueue: q,
		pendingTrxs: []Transaction{},
	}
}

func (t *Chain) appendEvent(e *Event) {
	t.Lock.Lock()
	defer t.Lock.Unlock()
	t.proofEvents[e.Proof] = e
	t.pendingEvents = append(t.pendingEvents, e)
}

func (t *Chain) SendToShardMember(origin string, payload []byte) {
	t.Lock.Lock()
	defer t.Lock.Unlock()
	s, ok := t.sockets.Get(origin)
	if ok {
		s.writePacket(payload)
	}
}

func (t *Chain) BroadcastInShard(payload []byte) {
	t.Lock.Lock()
	defer t.Lock.Unlock()
	for k, v := range t.sockets.Items() {
		if k == t.app.Id() {
			continue
		}
		v.writePacket(payload)
	}
}

func (t *Chain) Listen(port int) {
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

func (t *Chain) listenForPackets(socket *Socket) {
	defer func() {
		t.sockets.Remove(socket.Id)
		socket.Conn.Close()
	}()
	lenBuf := make([]byte, 4)
	buf := make([]byte, 1024)
	readCount := 0
	oldReadCount := 0
	for {
		_, err := socket.Conn.Read(lenBuf)
		if err != nil {
			fmt.Println(err)
			return
		}
		length := int(binary.BigEndian.Uint32(lenBuf))
		readData := make([]byte, length)
		for {
			readLength, err := socket.Conn.Read(buf)
			if err != nil {
				fmt.Println(err)
				return
			}
			oldReadCount = readCount
			readCount += readLength
			if readCount >= length {
				copy(readData[oldReadCount:], buf[:readLength-(readCount-length)])
				copy(buf[0:readCount-length], buf[readLength-(readCount-length):readLength])
				log.Println("packet received")
				log.Println(string(readData))
				future.Async(func() {
					socket.processPacket(strings.Split(socket.Conn.LocalAddr().String(), ":")[0], readData, uint32(length))
				}, false)
				readCount -= length
				break
			} else {
				copy(readData[oldReadCount:readCount], buf[:readLength])
			}
		}
	}
}

func (t *Chain) handleConnection(conn net.Conn) {
	connId := crypto.SecureUniqueString()
	socket := &Socket{Buffer: [][]byte{}, Conn: conn, app: t.app, Ack: true}
	t.sockets.Set(connId, socket)
	future.Async(func() {
		t.listenForPackets(socket)
	}, false)
}

func (t *Socket) writePacket(packet []byte) {

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
			packetLen := make([]byte, 4)
			binary.BigEndian.PutUint32(packetLen, uint32(len(t.Buffer[0])))
			_, err := t.Conn.Write(packetLen)
			if err != nil {
				t.Ack = true
				log.Println(err)
				return
			}
			_, err = t.Conn.Write(t.Buffer[0])
			if err != nil {
				t.Ack = true
				log.Println(err)
				return
			}
		}
	}
}

func (chainSocket *Socket) processPacket(origin string, packet []byte, packetLength uint32) {

	log.Println("received packet length: ", packetLength)

	pointer := uint32(1)

	if packet[0] == 0x01 {

		signature := ""
		data := ""

		log.Println("received consensus packet phase 1 from ", origin)

		tempBytes := make([]byte, 4)
		copy(tempBytes, packet[pointer:pointer+4])
		signatureLength := binary.BigEndian.Uint32(tempBytes)
		log.Println("signature length: ", signatureLength)
		pointer += 4
		if signatureLength > 0 {
			signature = string(packet[pointer : pointer+signatureLength])
			pointer += signatureLength
		}
		log.Println("signature: ", signature)

		tempBytes = make([]byte, 4)
		copy(tempBytes, packet[pointer:pointer+4])
		dataLength := binary.BigEndian.Uint32(tempBytes)
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

		if eventObj.Origin != origin {
			return
		}

		chainSocket.chain.appendEvent(eventObj)
		proofSign := chainSocket.chain.app.SignPacket([]byte(eventObj.Proof))

		proofSignBytes := []byte(proofSign)
		proofSignLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(proofSignLenBytes[:], uint32(len(proofSignBytes)))
		proofBytes := []byte(eventObj.Proof)
		proofLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(proofLenBytes[:], uint32(len(proofBytes)))

		updateLen := 1 + 4 + len(proofSignBytes) + 4 + len(proofBytes)

		update := make([]byte, updateLen)
		pointer := uint32(1)
		update[0] = 0x04
		copy(update[pointer:pointer+4], proofSignLenBytes)
		pointer += 4
		copy(update[pointer: pointer + uint32(len(proofSignBytes))], proofSignBytes)
		pointer += uint32(len(proofSignBytes))
		copy(update[pointer: pointer + 4], proofLenBytes)
		pointer += 4
		copy(update[pointer:pointer+uint32(len(proofBytes))], proofBytes)
		pointer += uint32(len(proofBytes))
		eventObj.MyUpdate = update

		responseLen := 1 + 4 + len(proofBytes)
		response := make([]byte, responseLen)
		response[0] = 0x02
		pointer = 1
		copy(response[pointer:pointer+4], proofLenBytes)
		pointer += 4
		copy(response[pointer:pointer+uint32(len(proofBytes))], proofBytes)
		pointer += uint32(len(proofBytes))

		chainSocket.chain.SendToShardMember(origin, response)

	} else if packet[0] == 0x02 {

		proof := ""

		log.Println("received consensus packet phase 2 from ", origin)

		tempBytes2 := make([]byte, 4)
		copy(tempBytes2, packet[pointer:pointer+4])
		dataLength := binary.BigEndian.Uint32(tempBytes2)
		log.Println("data length: ", dataLength)
		pointer += 4
		if dataLength > 0 {
			proof = string(packet[pointer : pointer+dataLength])
			pointer += dataLength
		}
		log.Println("proof: ", proof)

		done := chainSocket.chain.MemorizeResponseBacked(proof, origin)

		if !done {
			return
		}

		proofBytes := []byte(proof)
		proofLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(proofLenBytes, uint32(len(proofBytes)))

		reqLen := 1 + 4 + len(proofBytes)
		req := make([]byte, reqLen)
		pointer = uint32(1)
		req[0] = 0x03
		copy(req[pointer:pointer+4], proofLenBytes)
		pointer += 4
		copy(req[pointer:pointer+uint32(len(proofBytes))], proofBytes)
		pointer += uint32(len(proofBytes))

		chainSocket.chain.BroadcastInShard(req)

		chainSocket.chain.PushNewElection()
	} else if packet[0] == 0x03 {

		proof := ""

		log.Println("received consensus packet phase 3 from ", origin)

		tempBytes2 := make([]byte, 4)
		copy(tempBytes2, packet[pointer:pointer+4])
		dataLength := binary.BigEndian.Uint32(tempBytes2)
		log.Println("data length: ", dataLength)
		pointer += 4
		if dataLength > 0 {
			proof = string(packet[pointer : pointer+dataLength])
			pointer += dataLength
		}
		log.Println("proof: ", proof)

		chainSocket.chain.PushNewElection()

	} else if packet[0] == 0x04 {

		log.Println("received consensus packet phase 4 from ", origin)

		signature := ""
		vote := ""

		tempBytes := make([]byte, 4)
		copy(tempBytes, packet[pointer:pointer+4])
		pointer += 4
		signLength := binary.BigEndian.Uint32(tempBytes)
		log.Println("signature length: ", signLength)
		if signLength > 0 {
			signature = string(packet[pointer : pointer+signLength])
			pointer += signLength
		}
		log.Println("signature: ", signature)

		tempBytes2 := make([]byte, 4)
		copy(tempBytes2, packet[pointer:pointer+4])
		pointer += 4
		dataLength := binary.BigEndian.Uint32(tempBytes2)
		log.Println("data length: ", dataLength)
		if dataLength > 0 {
			vote = string(packet[pointer : pointer+dataLength])
			pointer += dataLength
		}
		log.Println("vote: ", vote)

		chainSocket.chain.VoteForNextEvent(origin, vote)

	} else if packet[0] == 0x05 {

		log.Println("received consensus packet phase 5 from ", origin)

		chainSocket.chain.NotifyElectorReady(origin)
	}
}

func (c *Chain) PushNewElection() {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.pendingBlockElections++
	if c.readyForNewElection {
		c.readyForNewElection = false
		c.ready = true
		c.cond_var_ <- 1
	}
}

func (c *Chain) NotifyElectorReady(origin string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.readyElectors[origin] = true
	if len(c.readyElectors) == (c.sockets.Count() - 1) {
		c.readyElectors = map[string]bool{}

		delete(c.proofEvents, c.chosenProof)
		eventIndex := 0
		for _, event := range c.pendingEvents {
			if event.Proof == c.chosenProof {
				break
			}
			eventIndex++
		}
		c.pendingEvents = append(c.pendingEvents[:eventIndex], c.pendingEvents[eventIndex+1:]...)
		c.chosenProof = ""

		c.ready = true
		c.cond_var_ <- 1
	}
}

func (c *Chain) VoteForNextEvent(origin string, eventProof string) {

	var choosenEvent *Event = nil
	done := false
	func() {
		c.Lock.Lock()
		defer c.Lock.Unlock()
		c.nextEventVotes[origin] = eventProof
		if len(c.nextEventVotes) == c.sockets.Count() {
			votes := map[string]int{}
			for _, vote := range c.nextEventVotes {
				if _, ok := votes[vote]; !ok {
					votes[vote] = 1
				} else {
					votes[vote] = votes[vote] + 1
				}
			}
			type Candidate struct {
				Proof string
				Votes int
			}
			sortedArr := []Candidate{}
			for proof, votes := range votes {
				sortedArr = append(sortedArr, Candidate{proof, votes})
			}
			slices.SortStableFunc(sortedArr, func(a Candidate, b Candidate) int { return a.Votes - b.Votes })
			c.nextEventVotes = map[string]string{}

			choosenEvent = c.GetEventByProof(sortedArr[0].Proof)
			c.chosenProof = sortedArr[0].Proof
			done = true
		}
	}()
	if done {
		c.nextBlockQueue.Put(choosenEvent)
		startNewElectionSignal := make([]byte, 1)
		startNewElectionSignal[0] = 0x05
		c.BroadcastInShard(startNewElectionSignal)
	}
}

func (c *Chain) GetEventByProof(proof string) *Event {
	return c.proofEvents[proof]
}

func openSocket(origin string, chain *Chain) bool {

	log.Println("connecting to chain socket server: ", origin)
	conn, err := net.Dial("tcp", origin+":8082")
	if err != nil {
		log.Println("Error connecting to server:", err)
		return false
	}
	log.Println("connected to the server..")
	chain.handleConnection(conn)
	return true
}

func (c *Chain) Run(port int) {

	future.Async(func() {
		for {
			haveTrxs := false
			func() {
				c.Lock.Lock()
				defer c.Lock.Unlock()
				if len(c.pendingTrxs) > 0 {
					haveTrxs = true
				}
			}()
			if haveTrxs {
				log.Println("creating new event...")
				var e *Event = nil
				func() {
					c.Lock.Lock()
					defer c.Lock.Unlock()
					proof := fmt.Sprintf("%d", time.Now().UnixMilli())
					e = &Event{Origin: c.app.Id(), Transactions: c.pendingTrxs, Proof: proof, MyUpdate: []byte{}}
					c.pendingTrxs = []Transaction{}
					c.pendingEvents = append(c.pendingEvents, e)
					c.proofEvents[e.Proof] = e
				}()
				dataStr, _ := json.Marshal(e)
				signature := c.app.SignPacket(dataStr)
				dataBytes := []byte(dataStr)
				dataLenBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(dataLenBytes, uint32(len(dataBytes)))
				signBytes := []byte(signature)
				signLenBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(signLenBytes, uint32(len(signBytes)))

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

				proofSign := c.app.SignPacket([]byte(e.Proof))
				proofSignBytes := []byte(proofSign)
				proofSignLenBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(proofSignLenBytes, uint32(len(proofSignBytes)))
				proofBytes := []byte(e.Proof)
				proofLenBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(proofLenBytes, uint32(len(proofBytes)))

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
	}, true)

	future.Async(func() {
		for {
			<-c.cond_var_
			func() {
				c.Lock.Lock()
				defer c.Lock.Unlock()
				c.ready = false
			}()
			if c.pendingBlockElections > 0 {
				e := c.pendingEvents[0]
				c.VoteForNextEvent(c.app.Id(), e.Proof)
				c.BroadcastInShard(e.MyUpdate)
				func() {
					c.Lock.Lock()
					defer c.Lock.Unlock()
					c.pendingBlockElections--
				}()
			} else {
				func() {
					c.Lock.Lock()
					defer c.Lock.Unlock()
					c.readyForNewElection = true
				}()
			}
		}
	}, true)

	future.Async(func() {
		for {
			blockRaw, _ := c.nextBlockQueue.Get()
			block := blockRaw.(*Event)
			for _, trx := range block.Transactions {
				log.Println("received transaction: ", trx.Typ, string(trx.Payload))
			}
		}
	}, true)

	future.Async(func() {
		log.Println("trying to connect to other peers...")
		peersArr := []string{
			"172.77.5.1",
			"172.77.5.2",
			"172.77.5.3",
		}
		completed := false
		for !completed {
			completed = true
			for _, peerAddress := range peersArr {
				log.Println("socket: ", peerAddress)
				if peerAddress == c.app.Id() {
					c.sockets.Set(peerAddress, &Socket{app: c.app, chain: c, Conn: nil, Buffer: [][]byte{}, Ack: true})
					continue
				}
				if peerAddress < c.app.Id() {
					continue
				}
				if c.sockets.Has(peerAddress) {
					continue
				}
				if !openSocket(peerAddress, c) {
					completed = false
					continue
				}
			}
			time.Sleep(time.Duration(1) * time.Second)
		}
	}, false)
}

func (c *Chain) SubmitTrx(t string, data []byte) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.pendingTrxs = append(c.pendingTrxs, Transaction{t, data})
}

func (c *Chain) MemorizeResponseBacked(proof string, origin string) bool {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	if e, ok := c.proofEvents[proof]; ok {
		e.backedResponses[origin] = true
		if len(e.backedResponses) == (c.sockets.Count() - 1) {
			return true
		}
	}
	return false
}

func (c *Chain) AddBackedProof(proof string, origin string, signedProof string) bool {
	// EVP_PKEY *pkey;
	// {
	//     std::lock_guard<std::mutex> lock(this->lock);
	//     pkey = this->shardPeers[origin]->pkey;
	// }
	// if (Utils::getInstance().verify_signature_rsa(pkey, proof, signedProof))
	if true {
		c.Lock.Lock()
		defer c.Lock.Unlock()
		if e, ok := c.proofEvents[proof]; ok {
			e.backedProofs[origin] = signedProof
			if len(e.backedProofs) == (c.sockets.Count() - 2) {
				if _, ok = e.backedProofs[e.Origin]; !ok {
					return true
				}
			}
		}
	}
	return false
}

func (c *Chain) RemoveConnection(origin string) {
	c.sockets.Remove(origin)
}
