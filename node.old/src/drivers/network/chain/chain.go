package chain

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"kasper/src/abstract/models/core"
	"kasper/src/shell/utils/future"
	"log"
	"math/rand"
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
	backedResponses map[string]Guarantee
	randomNums      map[string]int
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
	pipeline              func([][]byte)
}

type Ok struct {
	Proof  string
	RndNum int
}

func NewChain(core core.ICore) *Chain {
	m := cmap.New[*Socket]()
	q, _ := queues.NewLinkedBlockingQueue(1000)
	return &Chain{
		app:                   core,
		sockets:               &m,
		proofEvents:           map[string]*Event{},
		pendingEvents:         []*Event{},
		chosenProof:           "",
		pendingBlockElections: 0,
		readyForNewElection:   true,
		ready:                 false,
		cond_var_:             make(chan int, 10000),
		readyElectors:         map[string]bool{},
		nextEventVotes:        map[string]string{},
		nextBlockQueue:        q,
		pendingTrxs:           []Transaction{},
	}
}

func (t *Chain) appendEvent(e *Event) {
	t.Lock.Lock()
	defer t.Lock.Unlock()
	t.proofEvents[e.Proof] = e
	t.pendingEvents = append(t.pendingEvents, e)
}

func (t *Chain) SendToShardMember(origin string, payload []byte) {
	log.Println("sending packet to peer:", origin)
	s, ok := t.sockets.Get(origin)
	if ok {
		s.writePacket(payload)
	} else {
		log.Println("peer socket not found")
	}
}

func (t *Chain) BroadcastInShard(payload []byte) {
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
			t.handleConnection(conn)
		}
	}, true)
}

func (t *Chain) listenForPackets(socket *Socket) {
	defer func() {
		t.sockets.Remove(socket.Id)
		socket.Conn.Close()
	}()
	origin := strings.Split(socket.Conn.RemoteAddr().String(), ":")[0]
	lenBuf := make([]byte, 4)
	buf := make([]byte, 1024)
	nextBuf := make([]byte, 2048)
	readCount := 0
	oldReadCount := 0
	enough := false
	beginning := true
	length := 0
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

				oldReadCount = readCount
				readCount += readLength
				copy(nextBuf[remainedReadLength:remainedReadLength+readLength], buf[0:readLength])
				remainedReadLength += readLength

				log.Println(origin, nextBuf[0:readLength])

				log.Println(origin, "stat 1:", readLength, oldReadCount, readCount, remainedReadLength)
			}()
		}

		if beginning {
			if readCount >= 4 {
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
					log.Println(origin, "stat 3 step 4")
					future.Async(func() {
						socket.processPacket(origin, packet, lengthOfPacket)
					}, false)
					enough = true
					beginning = true

					log.Println(origin, "stat 3:", remainedReadLength, oldReadCount, readCount)
				}()
			} else {
				func() {
					socket.Lock.Lock()
					defer socket.Lock.Unlock()
					log.Println(origin, "stating stat 4...")

					copy(readData[oldReadCount:oldReadCount+(readCount-oldReadCount)], nextBuf[0:readCount-oldReadCount])
					remainedReadLength = 0
					enough = true

					log.Println(origin, "stat 4:", remainedReadLength)
				}()
			}
		}
	}
}

func (t *Chain) handleConnection(conn net.Conn) {
	socket := &Socket{Buffer: [][]byte{}, Conn: conn, app: t.app, chain: t, Ack: true}
	t.sockets.Set(strings.Split(conn.RemoteAddr().String(), ":")[0], socket)
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
			binary.LittleEndian.PutUint32(packetLen, uint32(len(t.Buffer[0])))
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

func (chainSocket *Socket) processPacket(origin string, packet []byte, packetLength int) {

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
	}()

	pointer := 1

	if packet[0] == 0x01 {

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

		log.Println("step 1")

		if eventObj.Origin != origin {
			log.Println("origins does not match :\n", eventObj.Origin, "\n", origin)
			return
		}

		log.Println("step 2")

		chainSocket.chain.appendEvent(eventObj)
		proofSign := chainSocket.chain.app.SignPacket([]byte(eventObj.Proof))

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

		okObj := Ok{Proof: eventObj.Proof, RndNum: rand.Intn(chainSocket.chain.sockets.Count())}
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

		chainSocket.chain.SendToShardMember(origin, response)

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

		done := chainSocket.chain.MemorizeResponseBacked(okObj.Proof, sign, okObj.RndNum, origin)

		if !done {
			return
		}

		e := chainSocket.chain.GetEventByProof(okObj.Proof)
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

		chainSocket.chain.BroadcastInShard(req)

		chainSocket.chain.PushNewElection()
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

		e := chainSocket.chain.GetEventByProof(m[chainSocket.app.Id()].Proof)
		rndNums := map[string]int{}
		for k, v := range m {
			rndNums[k] = v.RndNum
		}
		e.randomNums = rndNums

		chainSocket.chain.PushNewElection()

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

		chainSocket.chain.VoteForNextEvent(origin, vote)

	} else if packet[0] == 0x05 {

		log.Println("received consensus packet phase 5 from ", origin)

		chainSocket.chain.NotifyElectorReady(origin)
	}
}

func (c *Chain) PushNewElection() {
	push := false
	func() {
		c.Lock.Lock()
		defer c.Lock.Unlock()
		c.pendingBlockElections++
		if c.readyForNewElection {
			c.readyForNewElection = false
			c.ready = true
			push = true
		}
	}()
	if push {
		c.cond_var_ <- 1
	}
}

func (c *Chain) NotifyElectorReady(origin string) {
	push := false
	func() {
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
			push = true
		}
	}()
	if push {
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
				c.chosenProof = chosenProofRes
			} else {
				choosenEvent = c.GetEventByProof(sortedArr[0].Proof)
				c.chosenProof = sortedArr[0].Proof
			}
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

	c.Listen(port)

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
					e = &Event{
						Origin:          c.app.Id(),
						backedResponses: map[string]Guarantee{},
						backedProofs:    map[string]string{},
						randomNums:      map[string]int{},
						Transactions:    c.pendingTrxs,
						Proof:           proof,
						MyUpdate:        []byte{},
					}
					c.pendingTrxs = []Transaction{}
					c.pendingEvents = append(c.pendingEvents, e)
					c.proofEvents[e.Proof] = e
				}()
				dataStr, _ := json.Marshal(e)
				signature := c.app.SignPacket(dataStr)
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

				proofSign := c.app.SignPacket([]byte(e.Proof))
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
			pipelinePacket := [][]byte{}
			for _, trx := range block.Transactions {
				log.Println("received transaction: ", trx.Typ, string(trx.Payload))
				pipelinePacket = append(pipelinePacket, trx.Payload)
			}
			c.pipeline(pipelinePacket)
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

func (c *Chain) SubmitTrx(typ string, payload []byte) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.pendingTrxs = append(c.pendingTrxs, Transaction{Typ: typ, Payload: payload})
}

func (c *Chain) MemorizeResponseBacked(proof string, signature string, rndNum int, origin string) bool {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	if e, ok := c.proofEvents[proof]; ok {
		e.backedResponses[origin] = Guarantee{Proof: proof, Sign: signature}
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

func (c *Chain) RegisterPipeline(pipeline func([][]byte)) {
	c.pipeline = pipeline
}

func (c *Chain) Peers() []string {
	return c.sockets.Keys()
}