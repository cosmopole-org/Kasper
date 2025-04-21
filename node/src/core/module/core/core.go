package module_core

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"kasper/src/abstract/adapters/tools"
	"kasper/src/abstract/models"
	"kasper/src/abstract/models/action"
	"kasper/src/abstract/models/chain"
	"kasper/src/abstract/models/info"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/models/update"
	"kasper/src/babble"
	"kasper/src/core/module/actor/model/l2"
	module_trx "kasper/src/core/module/actor/model/trx"
	mach_model "kasper/src/shell/machiner/model"
	"kasper/src/shell/utils/crypto"
	"kasper/src/shell/utils/future"
	"log"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	module_model "kasper/src/shell/layer2/model"

	"kasper/src/proxy/inmem"
)

type Core struct {
	lock           sync.Mutex
	id             string
	tools          *tools.Tools
	gods           []string
	chain          chan any
	chainCallbacks map[string]*chain.ChainCallback
	babbleInst     *babble.Babble
	Ip             string
	elections      []chain.Election
	elecReg        bool
	elecStarter    string
	elecStartTime  int64
	executors      map[string]bool
	appPendingTrxs []*worker.Trx
	actionStore    map[string]action.IAction
}

var MAX_VALIDATOR_COUNT = 5

func NewCore(_ string) *Core {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr).IP.String()
	id := localAddr
	execs := map[string]bool{}
	execs["172.77.5.1"] = true
	execs["172.77.5.2"] = true
	return &Core{
		id:             id,
		gods:           make([]string, 0),
		chain:          nil,
		chainCallbacks: map[string]*chain.ChainCallback{},
		babbleInst:     nil,
		Ip:             localAddr,
		elections:      nil,
		elecReg:        false,
		executors:      execs,
	}
}

func (c *Core) ModifyStateSecurly(readonly bool, info info.IInfo, fn func (l2.State)) {
	trx := module_trx.NewTrx(c, c.Tools().Storage, readonly)
	defer trx.Commit()
	s := l2.NewState(info, trx)
	fn(s)
}

func (c *Core) ModifyState(readonly bool, fn func (trx.ITrx)) {
	trx := module_trx.NewTrx(c, c.Tools().Storage, readonly)
	defer trx.Commit()
	fn(trx)
}

func (c *Core) Tools() *tools.Tools {
	return c.tools
}

func (c *Core) Id() string {
	return c.id
}

func (c *Core) Chain() *babble.Babble {
	return c.babbleInst
}

func (c *Core) Gods() []string {
	return c.gods
}

func (c *Core) IpAddr() string {
	return c.Ip
}

func (c *Core) AppPendingTrxs() {
	elpisTrxs := []*worker.Trx{}
	wasmTrxs := []*worker.Trx{}
	for _, trx := range c.appPendingTrxs {
		if trx.Runtime == "elpis" {
			elpisTrxs = append(elpisTrxs, trx)
		} else if trx.Runtime == "wasm" {
			wasmTrxs = append(wasmTrxs, trx)
		}
	}
	if len(elpisTrxs) > 0 {
		c.Tools().Elpis.ExecuteChainTrxsGroup(elpisTrxs)
	}
	if len(wasmTrxs) > 0 {
		c.Tools().Wasm.ExecuteChainTrxsGroup(wasmTrxs)
	}
	c.appPendingTrxs = []*worker.Trx{}
}

func (c *Core) ClearAppPendingTrxs() {
	c.appPendingTrxs = []*worker.Trx{}
}

func (c *Core) ExecAppletRequestOnChain(topicId string, machineId string, key string, packet []byte, userId string, callback func([]byte, int, error)) {
	c.lock.Lock()
	defer c.lock.Unlock()
	callbackId := crypto.SecureUniqueString()
	c.chainCallbacks[callbackId] = &chain.ChainCallback{Fn: callback, Executors: map[string]bool{}, Responses: map[string]string{}}
	trx := c.ModifyState(true)
	defer trx.Commit()
	vm := mach_model.Vm{MachineId: machineId}.Pull(trx)
	future.Async(func() {
		c.chain <- chain.ChainPacket{Type: "request", Meta: map[string]any{"requester": c.Ip, "origin": c.id, "requestId": callbackId, "isBase": false, "runtime": vm.Runtime, "userId": userId, "machineId": machineId, "topicId": topicId}, Key: key, Payload: packet, Effects: models.Effects{DbUpdates: []models.Update{}}}
	}, false)
}

func (c *Core) ExecBaseRequestOnChain(key string, packet any, layer int, token string, callback func([]byte, int, error)) {
	c.lock.Lock()
	defer c.lock.Unlock()
	callbackId := crypto.SecureUniqueString()
	c.chainCallbacks[callbackId] = &chain.ChainCallback{Fn: callback, Executors: map[string]bool{}, Responses: map[string]string{}}
	serialized, err := json.Marshal(packet)
	if err == nil {
		future.Async(func() {
			c.chain <- chain.ChainPacket{Type: "request", Meta: map[string]any{"requester": c.Ip, "origin": c.id, "requestId": callbackId, "isBase": true, "layer": layer, "token": token}, Key: key, Payload: serialized, Effects: models.Effects{DbUpdates: []models.Update{}}}
		}, false)
	} else {
		log.Println(err)
	}
}

func (c *Core) ExecBaseResponseOnChain(callbackId string, packet any, resCode int, e string, updates []update.Update) {
	serialized, err := json.Marshal(packet)
	if err == nil {
		future.Async(func() {
			c.chain <- chain.ChainPacket{Type: "response", Meta: map[string]any{"executor": c.Ip, "requestId": callbackId, "isBase": true, "responseCode": resCode, "error": e}, Payload: serialized, Effects: chain.Effects{DbUpdates: updates}}
		}, false)
	} else {
		log.Println(err)
	}
}

func (c *Core) ExecAppletResponseOnChain(callbackId string, packet []byte, resCode int, e string, updates []update.Update) {
	future.Async(func() {
		c.chain <- chain.ChainPacket{Type: "response", Meta: map[string]any{"executor": c.Ip, "requestId": callbackId, "isBase": false, "responseCode": resCode, "error": e}, Payload: packet, Effects: chain.Effects{DbUpdates: updates}}
	}, false)
}

func (c *Core) OnChainPacket(packet chain.ChainPacket) {
	c.lock.Lock()
	defer c.lock.Unlock()
	switch packet.Type {
	case "election":
		{
			if packet.Key == "choose-validator" {
				phaseRaw, ok := packet.Meta["phase"]
				if !ok {
					return
				}
				phase, ok2 := phaseRaw.(string)
				if !ok2 {
					return
				}
				voterRaw, ok := packet.Meta["voter"]
				if !ok {
					return
				}
				voter, ok2 := voterRaw.(string)
				if !ok2 {
					return
				}
				if c.elections == nil {
					c.elections = []chain.Election{}
				}
				if phase == "start-reg" {
					c.elecReg = true
					c.elecStarter = voter
					c.elecStartTime = time.Now().UnixMilli()
					for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
						c.elections = append(c.elections, chain.Election{Participants: map[string]bool{}, Commits: map[string][]byte{}, Reveals: map[string]string{}})
					}
					future.Async(func() {
						c.chain <- chain.ChainPacket{
							Type:    "election",
							Key:     "choose-validator",
							Meta:    map[string]any{"phase": "register", "voter": c.Ip},
							Payload: []byte("{}"),
						}
					}, false)
					if voter == c.Ip {
						future.Async(func() {
							time.Sleep(time.Duration(10) * time.Second)
							c.chain <- chain.ChainPacket{
								Type:    "election",
								Key:     "choose-validator",
								Meta:    map[string]any{"phase": "end-reg", "voter": c.Ip},
								Payload: []byte("{}"),
							}
						}, false)
					}
				} else if phase == "end-reg" {
					if c.elecStarter == voter && ((time.Now().UnixMilli() - c.elecStartTime) > 8000) {
						c.elecReg = false
						payload := [][]byte{}
						nodeCount := c.babbleInst.Peers.Len()
						hasher := sha256.New()
						for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
							r := fmt.Sprintf("%d", rand.Intn(nodeCount))
							c.elections[i].MyNum = r
							hasher.Write([]byte(r))
							bs := hasher.Sum(nil)
							payload = append(payload, bs)
						}
						data, _ := json.Marshal(payload)
						future.Async(func() {
							c.chain <- chain.ChainPacket{
								Type:    "election",
								Key:     "choose-validator",
								Meta:    map[string]any{"phase": "commit", "voter": c.Ip},
								Payload: data,
							}
						}, false)
					}
				} else if phase == "register" {
					if c.elecReg {
						for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
							c.elections[i].Participants[voter] = true
						}
					}
				} else if phase == "commit" {
					votes := [][]byte{}
					e := json.Unmarshal(packet.Payload, &votes)
					if e != nil {
						return
					}
					if len(votes) < MAX_VALIDATOR_COUNT {
						return
					}
					if c.elections[0].Participants[voter] {
						for i := 0; i < min(MAX_VALIDATOR_COUNT, c.babbleInst.Peers.Len()); i++ {
							c.elections[i].Commits[voter] = votes[i]
						}
						if len(c.elections[0].Commits) == len(c.elections[0].Participants) {
							myReveals := []string{}
							for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
								myReveals = append(myReveals, c.elections[i].MyNum)
							}
							data, _ := json.Marshal(myReveals)
							future.Async(func() {
								c.chain <- chain.ChainPacket{
									Type:    "election",
									Key:     "choose-validator",
									Meta:    map[string]any{"phase": "reveal", "voter": c.Ip},
									Payload: data,
								}
							}, false)
						}
					}
				} else if phase == "reveal" {
					votes := []string{}
					e := json.Unmarshal(packet.Payload, &votes)
					if e != nil {
						return
					}
					if len(votes) < MAX_VALIDATOR_COUNT {
						return
					}
					if c.elections[0].Participants[voter] {
						for i := 0; i < min(MAX_VALIDATOR_COUNT, c.babbleInst.Peers.Len()); i++ {
							c.elections[i].Reveals[voter] = votes[i]
						}
						if len(c.elections[0].Reveals) == len(c.elections[0].Participants) {
							c.executors = map[string]bool{}
							nodesArr := []string{}
							for p := range c.elections[0].Participants {
								nodesArr = append(nodesArr, p)
							}
							sort.Strings(nodesArr)
							for _, elec := range c.elections[0:min(MAX_VALIDATOR_COUNT, len(nodesArr))] {
								res := -1
								first := true
								for v := range elec.Participants {
									hasher := sha256.New()
									commit := elec.Commits[v]
									reveal := elec.Reveals[v]
									hasher.Write([]byte(reveal))
									bs := hasher.Sum(nil)
									if !bytes.Equal(bs, commit) {
										continue
									}
									num, e := strconv.ParseInt(reveal, 10, 32)
									if e != nil {
										continue
									}
									if first {
										first = false
										res = int(num)
									} else {
										res ^= int(num)
									}
								}
								result := res % len(nodesArr)
								candidate := nodesArr[result]
								c.executors[candidate] = true
								nodesArr = append(nodesArr[:result], nodesArr[result+1:]...)
								c.elections = nil
							}
						}
					}
				}
			}
			break
		}
	case "request":
		{
			requesterRaw, ok := packet.Meta["requester"]
			if !ok {
				return
			}
			requester, ok2 := requesterRaw.(string)
			if !ok2 {
				return
			}
			requestIdRaw, ok := packet.Meta["requestId"]
			if !ok {
				return
			}
			requestId, ok2 := requestIdRaw.(string)
			if !ok2 {
				return
			}
			execs := map[string]bool{}
			for k, v := range c.executors {
				execs[k] = v
			}
			if requester == c.Ip {
				c.chainCallbacks[requestId].Executors = execs
			} else {
				c.chainCallbacks[requestId] = &chain.ChainCallback{Fn: nil, Executors: execs, Responses: map[string]string{}}
			}
			if !c.executors[c.Ip] {
				return
			}
			isBaseRaw, ok := packet.Meta["isBase"]
			if !ok {
				return
			}
			isBase, ok2 := isBaseRaw.(bool)
			if !ok2 {
				return
			}
			if isBase {
				layerRaw, ok := packet.Meta["layer"]
				if !ok {
					return
				}
				layer, ok2 := layerRaw.(float64)
				if !ok2 {
					return
				}
				originRaw, ok := packet.Meta["origin"]
				if !ok {
					return
				}
				origin, ok2 := originRaw.(string)
				if !ok2 {
					return
				}
				tokenRaw, ok := packet.Meta["token"]
				if !ok {
					return
				}
				token, ok2 := tokenRaw.(string)
				if !ok2 {
					return
				}
				action := c.actionStore[packet.Key]
				if action == nil {
					return
				}
				var input models.IInput
				i, err2 := action.(action.ISecureAction).ParseInput("fed", string(packet.Payload))
				if err2 != nil {
					log.Println(err2)
					c.ExecBaseResponseOnChain(requestId, abstract.EmptyPayload{}, 400, "input parsing error", []abstract.Update{}, []abstract.CacheUpdate{})
					return
				}
				input = i
				action.(abstract.ISecureAction).SecurlyActChain(l, token, requestId, input, origin)
			} else {
				userIdRaw, ok := packet.Meta["userId"]
				if !ok {
					return
				}
				userId, ok2 := userIdRaw.(string)
				if !ok2 {
					return
				}
				machineIdRaw, ok := packet.Meta["machineId"]
				if !ok {
					return
				}
				machineId, ok2 := machineIdRaw.(string)
				if !ok2 {
					return
				}
				runtimeRaw, ok := packet.Meta["runtime"]
				if !ok {
					return
				}
				runtimeId, ok2 := runtimeRaw.(string)
				if !ok2 {
					return
				}
				c.appPendingTrxs = append(c.appPendingTrxs, &worker.Trx{CallbackId: requestId, Runtime: runtimeId, UserId: userId, MachineId: machineId, Key: packet.Key, Payload: string(packet.Payload)})
			}
			break
		}
	case "response":
		{
			execitorAddrRaw, ok := packet.Meta["executor"]
			if !ok {
				return
			}
			execitorAddr, ok2 := execitorAddrRaw.(string)
			if !ok2 {
				return
			}
			callbackIdRaw, ok := packet.Meta["requestId"]
			if !ok {
				return
			}
			callbackId, ok2 := callbackIdRaw.(string)
			if !ok2 {
				return
			}
			callback, ok3 := c.chainCallbacks[callbackId]
			if ok3 {
				if !callback.Executors[execitorAddr] {
					return
				}
				str, _ := json.Marshal(ResponseHolder{Payload: packet.Payload, Effects: packet.Effects})
				callback.Responses[execitorAddr] = string(str)
				if len(callback.Responses) < len(callback.Executors) {
					return
				}
				temp := ""
				for _, res := range callback.Responses {
					if temp == "" {
						temp = res
					} else if res != temp {
						temp = ""
						break
					}
				}
				if temp == "" {
					return
				}
				if !callback.Executors[c.Ip] {
					tb := abstract.UseToolbox[*module_model.ToolboxL2](c.Get(2).Tools())
					kvstoreKeyword := "kvstore: "
					tb.Storage().DoTrx(func(trx adapters.ITrx) error {
						for _, ef := range packet.Effects.DbUpdates {
							if (len(ef.Data) > len(kvstoreKeyword)) && (ef.Data[0:len(kvstoreKeyword)] == kvstoreKeyword) {
								tb.Wasm().ExecuteChainEffects(ef.Data[0:len(kvstoreKeyword)])
							} else {
								trx.Db().Exec(ef.Data)
							}
						}
						return nil
					})
					for _, ef := range packet.Effects.CacheUpdates {
						if ef.Typ == "put" {
							tb.Cache().Put(ef.Key, ef.Val)
						} else if ef.Typ == "del" {
							tb.Cache().Del(ef.Key)
						}
					}
				}
				delete(c.chainCallbacks, callbackId)
				if callback.Fn != nil {
					resCodeRaw, ok := packet.Meta["responseCode"]
					if !ok {
						return
					}
					resCode, ok2 := resCodeRaw.(float64)
					if !ok2 {
						return
					}
					errCodeRaw, ok3 := packet.Meta["error"]
					if !ok3 {
						return
					}
					errCode, ok4 := errCodeRaw.(string)
					if !ok4 {
						return
					}
					if errCode == "" {
						callback.Fn(packet.Payload, int(resCode), nil)
					} else {
						callback.Fn(packet.Payload, int(resCode), errors.New(errCode))
					}
				}
			}
			break
		}
	}
}

func (c *Core) NewHgHandler() *abstract.HgHandler {
	return &abstract.HgHandler{
		Sigma: c,
	}
}

func (c *Core) Load(gods []string, layers []abstract.ILayer, args []interface{}) {
	c.gods = gods
	c.layers = layers
	var output = args
	for i := len(layers) - 1; i >= 0; i-- {
		output = layers[i].BackFill(c, output...)
	}

	engine := output[0].(*babble.Babble)
	proxy := output[1].(*inmem.InmemProxy)

	c.chain = make(chan any, 1)
	future.Async(func() {
		for {
			op := <-c.chain
			serialized, err := json.Marshal(op)
			if err == nil {
				log.Println(string(serialized))
				proxy.SubmitTx(serialized)
			} else {
				log.Println(err)
			}
		}
	}, true)

	var sb abstract.IStateBuilder
	for i := 0; i < len(layers); i++ {
		sb = layers[i].InitSb(sb)
		if i > 0 {
			layers[i].ForFill(c, layers[i-1].Tools())
		} else {
			layers[i].ForFill(c)
		}
	}

	c.babbleInst = engine
}

func (c *Core) DoElection() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.ElecReg = true
	future.Async(func() {
		c.chain <- abstract.ChainPacket{
			Type:    "election",
			Key:     "choose-validator",
			Meta:    map[string]any{"phase": "start-reg", "voter": c.Ip},
			Payload: []byte("{}"),
		}
	}, false)
}

func (c *Core) Run() {

	future.Async(func() {
		c.babbleInst.Run()
	}, false)

	future.Async(func() {
		for {
			time.Sleep(time.Duration(1) * time.Second)
			minutes := time.Now().Minute()
			seconds := time.Now().Second()
			if (minutes == 0) && ((seconds >= 0) && (seconds <= 2)) {
				c.DoElection()
				time.Sleep(2 * time.Minute)
			}
		}
	}, true)
}
